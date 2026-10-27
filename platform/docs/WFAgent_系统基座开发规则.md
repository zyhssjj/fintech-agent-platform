# WFAgent 系统基座开发指南

> 科技金融智能体 —— 基座开发范围、规范与交付清单

---

## 一、基座的定位与核心价值

**一句话定义：** 基座 = 所有业务模块共用的基础设施代码，是一次写好、处处复用的公共底盘。没有基座，每个组员都会重新发明登录、权限、日志、文件上传、异步任务调度——重复造轮子不说，最后合并代码还会因为实现方式不同而大量冲突。

| 没有基座的后果 | 有基座之后 |
|--------------|-----------|
| 每人写一套登录逻辑 | 登录/权限/日志 → import 即用 |
| 接口返回格式不统一 | 接口格式 → 框架统一封装 |
| 权限校验各自实现 | 文件上传 → 一个函数搞定 |
| 文件上传路径混乱 | 异步任务 → 调接口即可 |
| 合并时大量冲突 | 模块开发 → 只写业务逻辑 |

---

## 二、基座开发范围（后端 + 前端）

### 2.1 后端基座（Go 语言）—— 必须在其他模块启动前完成

| # | 组件 | 优先级 | 交付内容 |
|---|------|--------|---------|
| 1 | **统一项目脚手架** | P0 | Go Module 初始化、目录结构规范、编译脚本（Makefile）、Dockerfile 模板、Git 分支规范 |
| 2 | **数据库连接与 ORM** | P0 | PostgreSQL 连接池（GORM 或 sqlx）、Redis 客户端初始化、连接配置 HotReload、数据库迁移工具（golang-migrate） |
| 3 | **认证与授权中间件** | P0 | JWT 双 Token 签发与校验中间件、11 种角色权限矩阵的实现框架、API 路由守卫（按角色+资源拦截）、Token 自动续期 |
| 4 | **统一响应格式** | P0 | 标准 JSON 响应结构体（code/message/data）、分页响应格式、错误码枚举定义、参数校验错误统一格式 |
| 5 | **审计日志中间件** | P0 | 操作日志自动记录中间件（谁、何时、做了什么、IP）、日志持久化到操作日志表、敏感操作脱敏 |
| 6 | **文件服务** | P0 | MinIO 客户端封装（上传/下载/预览/删除）、文件MD5去重、临时下载链接生成、文件类型白名单校验 |
| 7 | **异步任务队列** | P0 | 任务创建/查询/取消统一接口、任务状态机实现（pending→running→success/failed/cancelled）、任务进度推送（WebSocket/SSE）、死信队列与重试机制 |
| 8 | **通用工具库** | P0 | 加密工具（MD5/AES）、时间格式化、分页计算器、参数校验器、ID生成器（雪花算法） |
| 9 | **API 网关路由框架** | P0 | HTTP 路由注册机制、中间件链、CORS 跨域配置、请求限流（令牌桶）、微服务路由转发（Agent/Fund/Invest/Project/Exit） |
| 10 | **配置管理** | P1 | YAML/ENV 配置加载、多环境配置切换（dev/staging/prod）、配置热更新 |

> **基座交付的最终形态：** 一个可独立启动的空系统（能登录、能分配角色权限、能上传文件、能启动异步任务），但没有任何业务 Module——这个空系统就是所有组员的开发起点。

### 2.2 前端基座（React 19）—— 必须与后端基座同步完成

| # | 组件 | 优先级 | 交付内容 |
|---|------|--------|---------|
| 1 | **项目脚手架** | P0 | React 19 + Vite + TypeScript + Zustand 初始化、目录结构、ESLint/Prettier 配置、组件命名规范 |
| 2 | **布局框架** | P0 | 侧边栏菜单 + 顶栏 + 内容区布局、菜单根据角色权限动态渲染、面包屑导航、页面标签页（TabView） |
| 3 | **路由守卫** | P0 | React Router v7 配置、登录状态拦截（未登录跳转登录页）、角色权限拦截（无权页面跳转 403） |
| 4 | **HTTP 请求封装** | P0 | fetch 封装（Token 自动注入、401 自动刷新Token、统一错误处理、Loading 状态管理） |
| 5 | **登录/注册页面** | P0 | 登录表单、Token 存储（localStorage + Zustand）、密码修改、忘记密码流程 |
| 6 | **通用组件库** | P0 | BaseDataTable 表格组件（分页/排序/筛选）、FileUploadField 文件上传组件、通用弹窗、全局通知 |
| 7 | **状态管理** | P1 | 用户信息 Store、权限 Store、全局配置 Store、主题切换 |
| 8 | **通用页面** | P1 | 403/404/500 错误页、个人中心、操作日志查看页、系统通知中心 |

> **前端基座的交付形态：** 一个能登录、能根据角色显示不同菜单、能调用后端 API 的空壳系统。业务组员拿到后只需在指定目录下创建页面组件、配置路由和菜单即可。

---

## 三、代码组织规则（必须遵守，否则合并时会爆炸）

### 3.1 Git 分支策略

```
# 基座开发人的分支
main ─── develop ─── feature/base-auth        # JWT + 权限
                    └── feature/base-file       # 文件服务
                    └── feature/base-task       # 异步任务

# 业务组员的分支（每人独立，从 develop 拉出）
develop ─── feature/module-fund      # 募资管理模块（组员A）
        └── feature/module-invest    # 投资管理模块（组员B）
        └── feature/module-project   # 项目管理模块（组员C）
        └── feature/module-meeting   # 会议管理模块（组员D）
        └── feature/module-agent     # 智能体定义模块（组员A）
```

**铁律 1：** 基座合并到 develop 后，所有组员 **必须从最新的 develop 拉分支**，不得基于老版本开发。

**铁律 2：** 业务模块代码 **只能落在自己模块的目录下**，禁止修改基座目录下的任何文件（如需修改基座，提 MR 由基座负责人 review 后合并）。

**铁律 3：** 每人每天至少向自己的 feature 分支提交一次，每周至少向 develop 合并一次，避免长期分叉。

### 3.2 后端目录结构（Go）

```
wfagent/
├── cmd/server/main.go              # 入口文件
├── internal/
│   ├── base/                       # 【基座目录 — 仅供基座负责人修改】
│   │   ├── middleware/             # JWT认证、权限校验、审计日志、限流
│   │   ├── auth/                   # 登录、Token刷新、密码管理
│   │   ├── file/                   # MinIO 上传/下载/删除
│   │   ├── task/                   # 异步任务队列
│   │   ├── response/               # 统一响应格式
│   │   ├── config/                 # 配置加载
│   │   └── utils/                  # 通用工具
│   ├── module-fund/               # 【募资管理模块 — 组员A】
│   ├── module-invest/             # 【投资管理模块 — 组员B】
│   ├── module-project/            # 【项目管理模块 — 组员C】
│   ├── module-meeting/            # 【会议管理模块 — 组员D】
│   ├── module-agent/              # 【智能体定义模块】
│   └── module-exit/               # 【退出管理模块】
├── pkg/                            # 对外暴露的公共包
├── migrations/                     # 数据库迁移脚本
└── deploy/                         # Dockerfile、k8s配置
```

### 3.3 前端目录结构（React 19）

```
web/
├── src/
│   ├── base/                       # 【基座目录 — 仅供基座负责人修改】
│   │   ├── layout/                # 布局框架
│   │   ├── router/                # 路由守卫（React Router v7）
│   │   ├── stores/                # Zustand 状态管理
│   │   ├── api/                   # fetch 封装
│   │   ├── components/            # 通用组件（表格/上传）
│   │   └── views/                 # 通用页面（登录/403/404/个人中心）
│   ├── module-fund/              # 【募资管理 — 组员A】
│   ├── module-invest/            # 【投资管理 — 组员B】
│   ├── module-project/           # 【项目管理 — 组员C】
│   ├── module-meeting/           # 【会议管理 — 组员D】
│   └── module-agent/             # 【智能体模块】
├── public/
└── vite.config.ts
```

### 3.4 每个业务模块内部的标准模板

基座负责人提供一个空白模块模板（骨架代码），组员只需 copy 模板目录后填充业务代码：

**后端模块模板（Go）**

```
module-xxx/
├── handler/        # HTTP 处理器
│   └── handler.go
├── service/        # 业务逻辑
│   └── service.go
├── repository/     # 数据访问
│   └── repository.go
├── model/          # 数据模型
│   └── model.go
├── router.go       # 路由注册（需引入）
└── README.md       # 接口清单
```

**前端模块模板（React 19）**

```
module-xxx/
├── api/            # 接口请求
│   └── index.ts
├── views/          # 页面组件
│   ├── List.tsx
│   └── Detail.tsx
├── components/     # 模块专有组件
├── stores/         # 模块 Store（Zustand）
├── router.ts       # 模块路由（React Router）
└── README.md
```

---

## 四、强制开发规范（写在基座 README 中）

| # | 规范项 | 基座已实现 | 组员如何使用 |
|---|--------|----------|------------|
| 1 | **接口响应格式** | 已封装 | 所有 handler 必须调用 `response.Success(c, data)` 或 `response.Error(c, code, msg)`，禁止自己拼 JSON |
| 2 | **认证校验** | 已封装 | 路由组使用 `router.Use(middleware.Auth())`，需要特定角色时加 `middleware.RequireRole("投资经理")` |
| 3 | **审计日志** | 已封装 | 在 handler 中调用 `middleware.AuditLog(c, "创建项目", projectID)`，框架自动记录操作人、时间、IP |
| 4 | **文件上传** | 已封装 | 调用 `fileService.Upload(ctx, reader, filename, bucket)`，返回文件 URL 和 MD5 |
| 5 | **异步任务** | 已封装 | 调用 `taskService.Submit(ctx, taskType, payload)` 创建任务，通过 taskID 轮询状态 |
| 6 | **数据库迁移** | 已封装 | 每个模块在 `migrations/` 下新增自己的 SQL 文件，文件命名格式 `0012_module_xxx_create_table.up.sql` |
| 7 | **错误处理** | 已封装 | 使用预定义错误码，新增错误码需要在 `base/response/errors.go` 中注册（提 MR 给基座负责人） |
| 8 | **日志规范** | 已封装 | 统一使用 `slog`（Go标准库），日志级别：Debug/Info/Warn/Error。生产环境禁止 Println |
| 9 | **API 文档** | 已封装 | 使用 Swagger 注解，每个接口必须写 `@Summary` `@Param` `@Success` 注释 |
| 10 | **前端 HTTP 请求** | 已封装 | 在模块 `api/index.ts` 中 `import { request } from '@/base/api'`，直接调 `request.get('/v1/projects')` 即可 |

---



## 五、组员上手检查清单（基座交付后每人逐项确认）

1. 从 develop 分支拉自己的 feature 分支
2. 在本地启动 PostgreSQL + Redis + MinIO（docker-compose 一键启动，基座已提供）
3. 运行基座的集成测试（`make test-base`），确认全部通过
4. 复制空白模块模板到自己的目录（`cp -r template/module-xxx internal/module-fund`）
5. 在 `migrations/` 下创建自己的建表 SQL（编号递增，不与其他人冲突）
6. 在自己模块的 `router.go` 中注册路由，启动服务确认路由生效
7. 写一个最简单的 CRUD 接口（如"创建基金"），从前端到后端全链路跑通
8. 确认自己在 `base/` 目录下**没有修改任何文件**（如有需要，提交 MR）

---

## 六、常见踩坑预警

| # | 坑 | 后果 | 正确做法 |
|---|----|------|---------|
| 1 | 组员不拉最新 develop 就开始开发 | 合并时发现基座 API 变了，大量冲突 | **每天 git pull origin develop** |
| 2 | 组员自己实现了"登录"或"权限" | 两套认证体系并存，安全漏洞 | 基座已提供，直接 import 使用，**绝对禁止自己写** |
| 3 | 不写 API 文档 | 前端组员不知道接口参数格式 | 每个接口必须写 Swagger 注解 |
| 4 | 文件直接存本地磁盘 | 服务器迁移时文件丢失 | 统一通过基座的 fileService 写入 MinIO |
| 5 | 模块间循环依赖 | 编译失败，模块边界模糊 | 模块间通过 API 调用（HTTP），不直接 import |
| 6 | 数据库迁移脚本编号冲突 | migration 执行失败 | 基座负责人分配编号区间（如募资 0010-0019，投资 0020-0029） |
