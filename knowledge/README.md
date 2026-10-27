# 科技金融企业知识库

面向航科君富、膜天君富、聚元私募业务资料的独立 RAG 服务。它提供文档入库、模型配置、混合检索、基于证据的问答和内部 API；其他系统通过服务端调用，不在浏览器暴露内部凭证。本次 V2 仅改造本独立服务，不修改科技金融主系统。

## V2 三级目录

- 航科君富：`beijing_hangke`，38 个三级事项。
- 膜天君富：`beijing_motian`，34 个三级事项。
- 聚元私募：`neimeng_juyuan`，39 个三级事项。
- 唯一依据为批准的 Excel；`config/catalog_v2.json` 保留各平台独立目录、二级释义、归类标准和 488 条相关文件记录。
- 更新配置：`python -m scripts.import_catalog_v2 "V2-知识库目录搭建-3个平台.xlsx"`，随后重新构建应用镜像。运行时无需安装 Excel 库。

支持 `PDF`、`DOCX`、`Markdown`、`TXT` 长文档。上传必须选择平台和完整三级叶子；原始文件按平台/叶子隔离。相关文件清单是归类提示，不限制文件名或要求全量上传。检索、问答和文档中心均支持一级、二级、三级前缀筛选；旧粗目录不再接受上传。

## 启动

1. 从 `.env.example` 创建 `.env`，填写 LLM、Embedding 和内部 API Key。重排模型可选，未配置时自动使用 BM25 与向量融合排序。
2. 打开 Docker Desktop。
3. Windows 运行 `start.bat --build`，macOS/Linux 运行 `./start.sh --build`。
4. 打开 `http://127.0.0.1:8201`，健康检查为 `http://127.0.0.1:8201/health`。

应用使用 `8201`，Elasticsearch 使用 `9202`，与参考项目隔离。运行数据保存在 `runtime/`，模型密钥只保存在本机 `.env` 和运行时加密配置中。

## 内部 API

- `POST /api/v1/retrieve`：返回筛选后的检索证据。
- `POST /api/v1/answer`：基于检索证据生成回答和来源。

请求需携带内部 API Key。参数包括 `query`、`company_code`、`domain_paths`、`top_k`。回答只依据命中文档；资料不足时应明确说明。

管理控制台可为每个智能体凭证按“平台 → 一级目录 → 二级目录 → 三级目录”分配权限。权限最终保存为三级叶子清单；调用方不传 `domain_paths` 时仅检索该凭证已授权的三级目录，显式请求未授权目录或平台时返回 `403`。旧版凭证继续按原有平台和领域字段兼容执行。

用户管理支持为一个用户绑定多个智能体凭证，并为用户生成独立的 `wk_user_...` API Key。用户调用 `/api/v1/retrieve` 时，检索范围是所有有效绑定凭证知识域的并集；调用 `/api/v1/answer` 时，只合并具有回答能力的凭证及其知识域，避免将回答能力扩大到仅允许检索的目录。凭证被停用、删除或从用户解绑后，用户权限立即收缩；用户密钥仅在创建或重置时显示一次。

目录树：`GET /api/companies/{company_code}/tree`；文档筛选：`GET /api/documents?company_code=beijing_hangke&domain_paths=基金层面`。

请求示例（`domain_paths` 可传任一级，省略表示当前平台全部资料）：

```json
{"query":"基金产品设计需要哪些资料？","company_code":"beijing_hangke","domain_paths":["基金层面/基金设立/基金产品设计"],"top_k":8}
```

## 清理与验证

日常启动不会清空资料。旧资料重置是一次性人工操作：先 `docker compose stop app`，再执行 `docker compose run --rm --no-deps app python -m scripts.reset_catalog_v2 --confirm-clear-old-documents`，最后 `docker compose up -d --no-build`。仅清空 `kb_finance_doc` 与 `data/finance`，保留运行配置和审计；请勿在有入库任务运行时重置，也勿删除整个 `runtime`。

完整逐条验收结果见 `V2改造验收清单.md`。Python 回归测试自带临时资料且自动清理：

```text
docker cp tests/. fintechkb-app:/app/tests/
docker exec -e KNOWLEDGE_TEST_BASE_URL=http://127.0.0.1:8000 fintechkb-app python -m unittest discover -s tests -v
node tests/test_catalog_controls.cjs
```

正常部署使用 `docker compose up -d --build`。外网无法下载基础镜像时，可使用已有依赖镜像执行代码层重建：

```text
docker tag fintech_knowledge-app fintechkb-deps:local
docker build --pull=false -f Dockerfile.offline -t fintech_knowledge-app .
docker compose up -d --no-build
```

离线构建要求本机已经存在安装了当前依赖的应用镜像；换电脑首次部署仍使用正常 Dockerfile。
