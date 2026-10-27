# WFAgent Frontend Base

本目录保留 React + Vite + TypeScript 前端工作台，并提供本地 Express 服务用于后端代理和 AI 报告生成。

## 分层

- `src/base/api`: 请求封装、Token 注入、统一响应拆包和接口契约。
- `src/base/stores`: 登录态和全局状态。
- `src/base/composables`: React hook 形式的业务数据封装。
- `src/base/components`: 通用组件。
- `src/module-*`: 按业务模块隔离 API、hooks、组件和页面。

## 配置

复制 `.env.example` 为 `.env`，至少确认：

```env
VITE_API_BASE_URL="/v1"
VITE_API_PROXY_TARGET="http://127.0.0.1:8080"
```

## DeepSeek API

AI 报告生成已切换为 DeepSeek/OpenAI-compatible 适配。服务端提供：

- `POST /api/ai/generate`
- `POST /api/deepseek/generate`
- `POST /api/gemini/generate`，仅作为历史兼容别名保留

云端 DeepSeek 示例：

```env
AI_PROVIDER=deepseek
DEEPSEEK_API_BASE_URL=https://api.deepseek.com/v1
DEEPSEEK_API_KEY=sk-...
DEEPSEEK_MODEL=deepseek-chat
```

本地 DeepSeek 预留适配示例，适用于 Ollama、vLLM、LM Studio 等 OpenAI-compatible 服务：

```env
AI_PROVIDER=local
LOCAL_DEEPSEEK_BASE_URL=http://127.0.0.1:11434/v1
LOCAL_DEEPSEEK_MODEL=deepseek-r1:7b
LOCAL_DEEPSEEK_API_KEY=local
```

未配置云端 Key 时，接口会返回内置演示报告，方便本地页面调试。

业务页面后续应优先调用模块 API，例如 `investmentApi.listProjects()`，不要在页面中直接拼接 `fetch`。
