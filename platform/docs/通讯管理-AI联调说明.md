# 通讯管理 AI 联调说明

本文档只覆盖本轮通讯管理模块新增的 AI 模板能力与前后端联调步骤，不改动现有 README。

## 1. 目标范围

本轮通讯管理已支持：

- 模板中心四入口
  - 选择模板
  - 新建模板
  - 录入模板
  - 历史模板
- Go 后端统一调用 Python AI 服务
- Python AI 服务读取 `ai/.env`
- 模板 AI 能力
  - 对话式新建模板
  - Word 模板录入解析
  - 函件正文建议

当前正式主流程仍以：

- 系统内生成
- 审批留痕
- 下载交付
- 人工外发

为准。

## 2. 目录结构

```text
fintechbase/
  ai/                         Python AI 服务
  backend/                    Go 后端
  frontend/                   React 前端
  docs/                       项目文档
```

## 3. AI 环境变量

先复制并填写：

```text
fintechbase/ai/.env
```

最少要填：

```env
AI_PROVIDER=openai
AI_BASE_URL=https://api.openai.com/v1
AI_API_KEY=你的密钥
AI_MODEL=gpt-5.4-mini
AI_TIMEOUT_MS=60000
AI_HOST=0.0.0.0
AI_PORT=9100
```

说明：

- `AI_BASE_URL` 填你实际使用的 OpenAI-compatible 地址
- `AI_API_KEY` 只放在 `ai/.env`，不要放到前端
- `AI_MODEL` 当前默认按 `gpt-5.4-mini`

## 4. 后端环境变量

后端 `.env` 需要能访问到 AI 服务：

```env
AI_SERVICE_BASE_URL=http://127.0.0.1:9100
AI_SERVICE_TIMEOUT=60s
```

如果 Python AI 服务不在本机 9100 端口，需要同步修改这个地址。

## 5. 启动顺序

建议按下面顺序启动。

### 5.1 启动 Python AI 服务

在 `fintechbase/ai` 目录执行：

```bash
pip install -r requirements.txt
python -m app.main
```

默认监听：

```text
http://127.0.0.1:9100
```

可用健康检查：

```text
GET http://127.0.0.1:9100/healthz
```

### 5.2 启动 Go 后端

在 `fintechbase/backend` 目录执行你们现有的后端启动命令。

后端会通过：

- `POST /template/chat`
- `POST /template/import`
- `POST /template/render-assist`

调用 Python AI 服务。

### 5.3 启动前端

在 `fintechbase/frontend` 目录执行：

```bash
npm install
npm run dev
```

若做生产构建可执行：

```bash
npm run build
```

## 6. 通讯管理手测步骤

建议用内部角色登录，例如运营经理账号。

### 6.1 新建模板

1. 打开 `通讯管理 -> 模板中心 -> 新建模板`
2. 输入模板目标、模板场景、最新输入
3. 点击 `生成 AI 草稿`
4. 检查：
   - 是否返回摘要
   - 是否返回待补充问题
   - 是否返回建议字段
   - 是否自动回填主题模板和 HTML 模板
5. 上传 `.docx` 模板
6. 点击 `保存为私有模板`

预期：

- 模板保存成功
- 模板出现在“我的私有模板”

### 6.2 录入模板

1. 打开 `通讯管理 -> 模板中心 -> 录入模板`
2. 上传一个 `.docx` 模板
3. 点击 `解析 Word 模板`
4. 检查：
   - 是否显示提取原文
   - 是否显示建议字段
   - 是否显示 AI 总结
5. 视情况调整字段列表
6. 点击 `保存为私有模板`

预期：

- 生成新的私有模板
- 模板带有 `import_file_name`

### 6.3 历史模板

1. 打开 `通讯管理 -> 模板中心 -> 历史模板`
2. 选择一个模板点击 `查看版本`

预期：

- 可看到版本号
- 可看到状态
- 可看到审批单号
- 可看到字段列表

### 6.4 私有模板提审

1. 在“历史模板”中选择一个私有模板
2. 点击 `提审升级为公共模板`
3. 去审批中心完成审批
4. 回到模板中心刷新

预期：

- 私有模板状态更新
- 新增一份 `已发布公共模板`

### 6.5 函件正文建议与创建

1. 打开 `通讯管理 -> 函件通知`
2. 选择模板
3. 选择发送邮箱
4. 选择 LP 或手工填写收件人
5. 点击 `AI 生成正文建议`
6. 点击 `生成预览`
7. 点击 `创建函件`

预期：

- AI 建议正常返回
- 函件预览可见
- 函件实例创建成功

## 7. 常见问题

### 7.1 `ai service is not configured`

说明后端已启动，但 Python AI 服务未配置或未启动。

检查：

- `ai/.env` 是否已填写
- Python AI 服务是否已启动
- 后端 `AI_SERVICE_BASE_URL` 是否指向正确地址

### 7.2 模板导入没有解析结果

检查：

- 上传文件是否为 `.docx`
- 文件内是否存在可提取文本
- Python AI 服务日志中是否有模型调用失败信息

### 7.3 前端没有显示 AI 建议

检查浏览器请求：

- `/v1/agents/communication/templates/ai/chat`
- `/v1/agents/communication/templates/ai/import`
- `/v1/agents/communication/templates/ai/render-assist`

若后端返回 500，优先看后端日志和 AI 服务日志。

## 8. 当前已验证项

已完成的本地校验：

```bash
go test -mod=mod ./internal/module-communication ./internal/base/ai ./internal/base/config
python -m compileall ai
npm run lint
npm run build
```

这些校验证明：

- Go 后端通讯管理与 AI client 可编译并通过定向测试
- Python AI 服务代码可通过语法编译
- 前端类型检查通过
- 前端生产构建通过
