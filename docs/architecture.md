# 架构与服务边界

## 为什么采用一个仓库、多个服务

科技金融主平台和知识库属于同一业务展示项目，但开发语言、依赖和数据存储不同。仓库归并只解决源码、配置示例和启动文档分散的问题，不把两个进程或数据库强行合并，也不引入 Git 子模块。克隆一次即可获取两部分源码。

## 请求链路

1. React 工作台通过统一请求层调用 Go 后端。
2. Go 后端处理 JWT、业务权限、通讯、审批与业务数据存储。
3. 139 使用 IMAP 收取邮件、SMTP 发送邮件，以邮箱客户端授权码认证。
4. Outlook 使用 OAuth 2.0 授权码流程获取 Token，通过 Microsoft Graph API 收发邮件。
5. 模板生成调用 `platform/ai` 的独立 Python 服务。
6. 知识库检索、问答由 Go 后端调用独立 FastAPI 服务，后者负责知识域授权、解析、切块、索引、混合检索和证据回答。

## 知识库共享密钥

本地直连的初始配置需要一致：

```text
platform/backend/.env
  KNOWLEDGE_SERVICE_BASE_URL=http://127.0.0.1:8201
  KNOWLEDGE_SERVICE_API_KEY=<shared-key>

knowledge/.env
  KNOWLEDGE_INTERNAL_API_KEY=<same-shared-key>
```

`scripts/init-env.ps1` 首次初始化时自动设置。后续可以改用知识库管理界面创建的受限 API Key，并设置明确的公司、知识域和检索/回答权限。密钥只配置在服务器侧。

## 部署边界

- 主平台使用 PostgreSQL、Redis、MinIO；知识库使用自己的 Elasticsearch 索引及运行数据。
- 当前示例为“Go / React / AI 在宿主机运行，知识库在 Docker 运行”，所以 Go 连接 `127.0.0.1:8201`。
- 如果将 Go 也放入 Docker，`127.0.0.1` 会指向 Go 容器本身，需要改为共享网络中的服务名或可达的宿主机地址。
- 源项目的本地配置、运行容器和业务文档没有随归并修改。新仓库是独立副本，不接管原项目的 GitLab remote。
