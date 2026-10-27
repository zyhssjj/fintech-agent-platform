# 外部知识库 API Go Demo

仅使用 Go 标准库，默认检索当前已有测试资料：

- 平台：膜天君富（`beijing_motian`）
- 知识域：`基金层面/基金设立/工商注册-会商`
- 接口：`POST /api/v1/retrieve`

运行：

```powershell
go run ./examples/external_api_demo/main.go
```

程序会提示输入 `wk_` 开头的 API Key。测试局域网服务：

```powershell
go run ./examples/external_api_demo/main.go -base-url http://192.168.1.253:8201
```

测试智能回答（凭证必须具有“智能回答”能力）：

```powershell
go run ./examples/external_api_demo/main.go -mode answer
```

指定其他平台或知识域：

```powershell
go run ./examples/external_api_demo/main.go -company beijing_motian -domain "基金层面/基金设立/工商注册-会商"
```

也可以通过环境变量提供配置：`KNOWLEDGE_BASE_URL`、`KNOWLEDGE_API_KEY`。不要把真实 API Key 写入源码或提交版本库。
