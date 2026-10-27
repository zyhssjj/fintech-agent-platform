import express from "express";
import "dotenv/config";
import os from "os";
import path from "path";
import { createServer as createViteServer } from "vite";

type AiProvider = "deepseek" | "local";

type GenerateReportRequest = {
  type?: "investment_report" | "valuation_test" | "meeting_minutes" | string;
  projectName?: string;
  stage?: string;
  description?: string;
  metrics?: string;
  customPrompt?: string;
};

type AiRuntimeConfig = {
  provider: AiProvider;
  baseUrl: string;
  apiKey?: string;
  model: string;
  timeoutMs: number;
};

type ChatCompletionResponse = {
  choices?: Array<{
    message?: {
      content?: string | null;
    };
  }>;
  error?: {
    message?: string;
  };
};

function getLocalIPv4Address(): string | null {
  const networkInterfaces = os.networkInterfaces();

  for (const addresses of Object.values(networkInterfaces)) {
    const address = addresses?.find((item) => item.family === "IPv4" && !item.internal);
    if (address) {
      return address.address;
    }
  }

  return null;
}

async function readRequestBody(req: express.Request): Promise<Buffer | undefined> {
  if (["GET", "HEAD"].includes(req.method)) {
    return undefined;
  }

  const chunks: Buffer[] = [];
  for await (const chunk of req) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  return chunks.length > 0 ? Buffer.concat(chunks) : undefined;
}

function normalizeBaseUrl(baseUrl: string): string {
  return baseUrl.replace(/\/+$/, "");
}

function resolveAiRuntimeConfig(): AiRuntimeConfig {
  const provider = (process.env.AI_PROVIDER || "deepseek").toLowerCase() === "local" ? "local" : "deepseek";
  const timeoutMs = Number(process.env.AI_API_TIMEOUT_MS || 60000);

  if (provider === "local") {
    return {
      provider,
      baseUrl: normalizeBaseUrl(process.env.LOCAL_DEEPSEEK_BASE_URL || "http://127.0.0.1:11434/v1"),
      apiKey: process.env.LOCAL_DEEPSEEK_API_KEY || "local",
      model: process.env.LOCAL_DEEPSEEK_MODEL || "deepseek-r1:7b",
      timeoutMs,
    };
  }

  return {
    provider,
    baseUrl: normalizeBaseUrl(process.env.DEEPSEEK_API_BASE_URL || "https://api.deepseek.com/v1"),
    apiKey: process.env.DEEPSEEK_API_KEY,
    model: process.env.DEEPSEEK_MODEL || "deepseek-chat",
    timeoutMs,
  };
}

function hasUsableAiConfig(config: AiRuntimeConfig): boolean {
  if (config.provider === "local") {
    return Boolean(config.baseUrl && config.model);
  }

  return Boolean(config.baseUrl && config.model && config.apiKey && config.apiKey !== "MY_DEEPSEEK_API_KEY");
}

function buildReportPrompt(payload: GenerateReportRequest): { systemPrompt: string; prompt: string } {
  const { type, projectName, stage, description, metrics, customPrompt } = payload;

  const systemPrompt = `你是 WFAgent 的科技金融智能分析助手，专长 PE/VC、私募股权、资产管理和投后管理数字化。
请始终使用专业中文回答，语气客观、精确、结构化。
输出使用清晰 Markdown，避免空泛营销话术，优先给出量化估计、投资逻辑、风险识别和合规约束。`;

  if (type === "investment_report") {
    return {
      systemPrompt,
      prompt: `请为以下科技金融项目生成一份投前投资可行性报告：
- 项目名称：${projectName || "未命名科技项目"}
- 融资阶段：${stage || "A 轮"}
- 项目描述：${description || "AI 芯片与高性能边缘计算"}
- 目标指标：${metrics || "年营收增长 > 50%，毛利率 > 60%"}

请包含：
1. 项目概览与核心亮点
2. 行业与市场格局分析
3. 财务与运营核心指标评价
4. 战略协同度与投资逻辑
5. 关键风险识别与缓释措施`,
    };
  }

  if (type === "valuation_test") {
    return {
      systemPrompt,
      prompt: `请对以下项目进行估值测算与敏感性情景分析：
- 项目名称：${projectName || "未命名科技项目"}
- 融资阶段：${stage || "B 轮"}
- 估算收入：${metrics || "5000 万人民币"}
- 补充信息：${description || "主要面临研发周期、供应链替代和国际环境风险"}

请输出：
1. DCF 与可比公司倍数法估值区间
2. 乐观、基准、保守三种敏感性情景
3. 关键风险因素和折现率调整建议`,
    };
  }

  if (type === "meeting_minutes") {
    return {
      systemPrompt,
      prompt: `请根据以下原始记录生成专业 LP/项目投委会会议纪要：
- 会议名称：${projectName || "WFAgent 投决会"}
- 参会人员：${stage || "全体管理合伙人及核心 LP 顾问委员"}
- 会议要点：${description || "讨论募资进展、出资率、投后项目估值变化与后续行动"}

请包含：
1. 会议概览
2. 战略更新与讨论要点
3. 会议决议
4. 后续行动计划与时间表`,
    };
  }

  return {
    systemPrompt,
    prompt: customPrompt || "请根据系统规则，生成一份科技金融全生命周期分析简报。",
  };
}

async function callChatCompletion(config: AiRuntimeConfig, systemPrompt: string, prompt: string): Promise<string> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), config.timeoutMs);

  try {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    if (config.apiKey) {
      headers.Authorization = `Bearer ${config.apiKey}`;
    }

    const response = await fetch(`${config.baseUrl}/chat/completions`, {
      method: "POST",
      headers,
      body: JSON.stringify({
        model: config.model,
        messages: [
          { role: "system", content: systemPrompt },
          { role: "user", content: prompt },
        ],
        temperature: Number(process.env.AI_TEMPERATURE || 0.2),
      }),
      signal: controller.signal,
    });

    const data = (await response.json().catch(() => ({}))) as ChatCompletionResponse;
    if (!response.ok) {
      throw new Error(data.error?.message || `AI upstream responded with HTTP ${response.status}`);
    }

    const text = data.choices?.[0]?.message?.content?.trim();
    if (!text) {
      throw new Error("AI upstream returned an empty response.");
    }

    return text;
  } finally {
    clearTimeout(timeout);
  }
}

function buildFallbackReport(payload: GenerateReportRequest): string {
  const projectName = payload.projectName || "星瀚半导体";
  const stage = payload.stage || "A 轮";
  const description = payload.description || "AI 芯片与高性能边缘计算";
  const metrics = payload.metrics || "年营收增长 > 50%，毛利率 > 60%";

  if (payload.type === "valuation_test") {
    return `# WFAgent 估值测算：${projectName}

**融资阶段：** ${stage}
**输入指标：** ${metrics}

## 估值区间
- DCF 基准估值：约 5.8 亿元人民币
- 可比公司 PS 区间：12.0x - 15.0x
- 综合建议区间：4.2 - 7.5 亿元人民币

## 情景分析
| 情景 | 核心假设 | 企业价值 |
| :--- | :--- | :--- |
| 乐观 | 收入高增、毛利率维持 60% 以上 | 7.5 亿元 |
| 基准 | 收入按计划释放，客户验证正常推进 | 5.8 亿元 |
| 保守 | 交付延期、供应链成本上行 | 4.2 亿元 |

## 风险调整
建议重点关注研发兑现、客户集中度、供应链替代与后续融资窗口。`;
  }

  if (payload.type === "meeting_minutes") {
    return `# WFAgent 会议纪要：${projectName}

## 会议概览
本次会议围绕 ${description} 展开，重点评估当前业务进展、资源协调事项与后续决策安排。

## 讨论要点
- 募资和出资节奏整体符合预期
- 投后重点项目需要加强里程碑跟踪
- 估值调整需结合最新订单、收入确认和审计材料

## 会议决议
- 启动下一轮尽调材料复核
- 由投资团队补充关键客户访谈
- 下次会议前提交风险清单与行动计划`;
  }

  return `# WFAgent 投资可行性报告：${projectName}

**融资阶段：** ${stage}
**项目描述：** ${description}
**核心指标：** ${metrics}

## 项目概览
项目处于科技金融重点关注赛道，具备技术壁垒、国产替代和产业协同潜力。

## 市场判断
所在细分市场受算力升级、产业数字化和供应链安全驱动，未来三年预计保持较高增速。

## 投资逻辑
- 技术路线具备差异化，适合通过资本和产业资源共同放大
- 商业化路径清晰，但需持续验证客户复购和规模交付能力
- 与基金现有产业资源存在潜在协同

## 关键风险
- 研发与量产节奏低于预期
- 客户集中度偏高
- 后续融资环境变化导致估值承压`;
}

async function startServer() {
  const app = express();
  const PORT = Number(process.env.PORT || 3000);
  const apiProxyTarget = normalizeBaseUrl(process.env.VITE_API_PROXY_TARGET || "http://127.0.0.1:8080");

  app.use("/v1", async (req, res) => {
    try {
      const headers = new Headers();
      const excludedHeaders = new Set([
        "host",
        "connection",
        "expect",
        "keep-alive",
        "proxy-authenticate",
        "proxy-authorization",
        "te",
        "trailer",
        "transfer-encoding",
        "upgrade",
      ]);
      Object.entries(req.headers).forEach(([key, value]) => {
        if (excludedHeaders.has(key.toLowerCase())) return;
        if (Array.isArray(value)) {
          value.forEach((item) => headers.append(key, item));
          return;
        }
        if (value !== undefined) {
          headers.set(key, value);
        }
      });

      const body = await readRequestBody(req);
      const upstream = await fetch(`${apiProxyTarget}${req.originalUrl}`, {
        method: req.method,
        headers,
        body,
        duplex: "half",
      } as RequestInit & { duplex: "half" });

      upstream.headers.forEach((value, key) => {
        if (key.toLowerCase() !== "content-encoding") {
          res.setHeader(key, value);
        }
      });
      res.status(upstream.status).send(Buffer.from(await upstream.arrayBuffer()));
    } catch (error) {
      console.error("Backend API proxy failed:", error);
      res.status(502).json({ code: 10003, message: "Backend API is unavailable. Please start backend first." });
    }
  });

  app.use(express.json());

  const generateReport = async (req: express.Request, res: express.Response) => {
    const payload = req.body as GenerateReportRequest;
    const config = resolveAiRuntimeConfig();
    const { systemPrompt, prompt } = buildReportPrompt(payload);

    try {
      if (hasUsableAiConfig(config)) {
        const text = await callChatCompletion(config, systemPrompt, prompt);
        res.json({ success: true, text, provider: config.provider, model: config.model });
        return;
      }

      console.log("DeepSeek API is not configured. Returning built-in demo financial content.");
      res.json({
        success: true,
        text: buildFallbackReport(payload),
        provider: "fallback",
        model: "demo",
      });
    } catch (error: any) {
      console.error("DeepSeek compatible AI call failed:", error);
      res.status(500).json({ success: false, error: error.message || "AI generation failed" });
    }
  };

  app.post("/api/ai/generate", generateReport);
  app.post("/api/deepseek/generate", generateReport);
  app.post("/api/gemini/generate", generateReport);

  if (process.env.NODE_ENV !== "production") {
    const vite = await createViteServer({
      server: { middlewareMode: true },
      appType: "spa",
    });
    app.use(vite.middlewares);
  } else {
    const distPath = path.join(process.cwd(), "dist");
    app.use(express.static(distPath));
    app.get("*", (req, res) => {
      res.sendFile(path.join(distPath, "index.html"));
    });
  }

  app.listen(PORT, "0.0.0.0", () => {
    const localIPv4Address = getLocalIPv4Address();

    console.log(`[WFAgent Server] Local: http://localhost:${PORT}`);
    console.log(`[WFAgent Server] AI provider: ${resolveAiRuntimeConfig().provider}`);
    if (localIPv4Address) {
      console.log(`[WFAgent Server] LAN: http://${localIPv4Address}:${PORT}`);
    } else {
      console.log("[WFAgent Server] LAN: No non-internal IPv4 address found");
    }
  });
}

startServer();
