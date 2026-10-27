# 科技金融业务主平台

此目录是主平台源码，独立企业知识库位于同级 `../knowledge/`。完整启动步骤见 [仓库首页](../README.md)。

## 已接入的主要模块

- `backend/internal/base`：JWT、权限、审计、文件与异步任务基座。
- `backend/internal/module-communication`：139 IMAP/SMTP、Outlook OAuth 2.0 / Graph、邮件同步、模板与附件。
- `backend/internal/module-approval`：审批相关业务。
- `backend/internal/module-knowledge`：服务端代理独立知识库的检索及问答 API。
- `frontend/src/module-agent/views`：通讯、审批、知识库界面。
- `ai/app`：独立的 Python AI 模板服务。

Go + Gin + GORM + PostgreSQL；前端是 React + TypeScript，不是 Vue。基金、投资、投后、会议、退出等其他目录包含基座或扩展骨架，不能据此认定所有业务均已完整实现。

## 开发命令

在 `backend/` 执行 `go run ./cmd/server`、`go test ./...`。

在 `frontend/` 执行 `npm ci`、`npm run dev`、`npm run lint`、`npm run build`。

在 `ai/` 安装 `requirements.txt` 后执行 `python -m app.main`。

本地依赖使用 `deploy/docker-compose.yml`。修改数据库密码时，同时更新 `backend/.env` 与 Compose 配置。请先初始化配置，并参考根目录说明连接知识库。
