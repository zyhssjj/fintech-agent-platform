"""科技金融企业知识库系统 - FastAPI 服务。

对外 API（机器凭证 Bearer 鉴权）：
    POST /api/v1/retrieve   返回检索证据（不调 LLM）
    POST /api/v1/answer     调用知识库 LLM 返回纯文本回答
内部/管理 API：
    GET/POST/PUT/DELETE /api/routing/rules   可编辑查询路由规则
    GET/PUT /api/models/...                    模型配置（llm/embedding/rerank）
    GET/POST/DELETE /api/credentials           机器凭证
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import secrets
import subprocess
import sys
import threading
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

from dotenv import load_dotenv
from fastapi import BackgroundTasks, FastAPI, Depends, File, Form, Header, HTTPException, Query, Request, UploadFile
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
from elasticsearch import Elasticsearch

from scripts import llm, model_config, retrieval
from scripts.company_catalog import COMPANIES, domain_metadata, domain_tree, expand_domains, get_company, get_leaf, public_companies, valid_domain
from elasticsearch.helpers import scan
from scripts.auth_store import AuthStore
from scripts.embeddings import CloudEmbeddingProvider, configured_provider
from scripts import rerank_api

load_dotenv()

PROJECT_DIR = Path(__file__).resolve().parents[1]
RUNTIME_DIR = PROJECT_DIR / "runtime"
STATIC_DIR = PROJECT_DIR / "static"
CREDENTIALS_PATH = RUNTIME_DIR / "credentials.json"
ROUTING_PATH = RUNTIME_DIR / "routing_rules.json"
INGESTION_JOBS_DIR = RUNTIME_DIR / "ingestion_jobs"
DOCUMENT_AUDIT_PATH = RUNTIME_DIR / "document_audit.jsonl"
DATA_DIR = PROJECT_DIR / "data"

RUNTIME_DIR.mkdir(parents=True, exist_ok=True)
INGESTION_JOBS_DIR.mkdir(parents=True, exist_ok=True)

app = FastAPI(title="科技金融企业知识库", version="0.1.0")


@app.middleware("http")
async def disable_console_asset_cache(request: Request, call_next):
    response = await call_next(request)
    if request.url.path == "/" or request.url.path.startswith("/static/"):
        response.headers["Cache-Control"] = "no-store, max-age=0"
    return response

_lock = threading.Lock()
_es: Elasticsearch | None = None
_emb = None
auth_store = AuthStore(RUNTIME_DIR)


@app.on_event("startup")
def initialize_finance_index() -> None:
    """Create the finance index on a fresh Elasticsearch volume before UI requests."""
    from scripts.create_indices import INDEX_DEFINITIONS, index_definition

    client = es()
    if not client.indices.exists(index="kb_finance_doc"):
        client.indices.create(
            index="kb_finance_doc",
            **index_definition(INDEX_DEFINITIONS["kb_finance_doc"]),
        )
        return
    properties = client.indices.get_mapping(index="kb_finance_doc")["kb_finance_doc"]["mappings"].get("properties", {})
    missing = {name: definition for name, definition in INDEX_DEFINITIONS["kb_finance_doc"].items() if name not in properties}
    if missing:
        client.indices.put_mapping(index="kb_finance_doc", properties=missing)


def es() -> Elasticsearch:
    global _es
    if _es is None:
        _es = Elasticsearch(os.getenv("ES_URL", "http://localhost:9202"))
    return _es


def emb():
    global _emb
    if _emb is None:
        _emb = configured_provider()
    return _emb


# ---------- JSON 存储 ----------

def _read_json(path: Path, default):
    if not path.exists():
        return default
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (json.JSONDecodeError, OSError):
        return default


def _write_json(path: Path, data) -> None:
    with _lock:
        path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")


def _append_document_audit(action: str, document_id: str, source: str = "") -> None:
    record = {
        "action": action,
        "document_id": document_id,
        "source": source,
        "at": datetime.now(timezone.utc).isoformat(),
    }
    with _lock:
        with DOCUMENT_AUDIT_PATH.open("a", encoding="utf-8") as handle:
            handle.write(json.dumps(record, ensure_ascii=False) + "\n")


# ---------- 机器凭证 ----------

def _credential_store() -> dict:
    return _read_json(CREDENTIALS_PATH, {"clients": []})


def _save_credentials(data: dict) -> None:
    _write_json(CREDENTIALS_PATH, data)


def _sha256(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def _new_api_key() -> str:
    return "wk_" + secrets.token_hex(24)


def _find_client_by_key(key: str):
    digest = _sha256(key)
    for client in _credential_store()["clients"]:
        if client.get("api_key_sha256") == digest:
            return client
    return None


def _credential_map() -> dict[str, dict]:
    return {client.get("id"): client for client in _credential_store()["clients"] if client.get("id")}


def _validated_credential_ids(raw_ids) -> list[str]:
    if raw_ids is None:
        return []
    if not isinstance(raw_ids, list):
        raise HTTPException(status_code=422, detail="凭证绑定必须是数组")
    selected = list(dict.fromkeys(str(item).strip() for item in raw_ids if str(item).strip()))
    missing = [item for item in selected if item not in _credential_map()]
    if missing:
        raise HTTPException(status_code=422, detail="包含不存在的接入凭证")
    return selected


def _effective_user_client(user: dict) -> dict:
    """Merge active credential permissions for a user without exposing source keys."""
    clients = _credential_map()
    now = datetime.now(timezone.utc)
    active = []
    for credential_id in user.get("credential_ids") or []:
        client = clients.get(credential_id)
        if not client or not client.get("enabled", True):
            continue
        expires = client.get("expires_at")
        if expires:
            try:
                expiry = datetime.fromisoformat(expires)
                if expiry.tzinfo is None:
                    expiry = expiry.replace(tzinfo=timezone.utc)
                if expiry <= now:
                    continue
            except ValueError:
                pass
        active.append(client)

    scopes_by_company: dict[str, list[str]] = {}
    answer_scopes_by_company: dict[str, list[str]] = {}
    for client in active:
        for code in COMPANIES:
            paths = _client_domain_paths(client, code)
            if paths:
                scopes_by_company.setdefault(code, []).extend(paths)
                if client.get("can_answer", False):
                    answer_scopes_by_company.setdefault(code, []).extend(paths)
    knowledge_scopes = [
        {"company_code": code, "domain_paths": list(dict.fromkeys(paths))}
        for code, paths in scopes_by_company.items()
    ]
    limits = [max(1, int(client.get("rate_limit_per_min", 60))) for client in active]
    return {
        "id": "user-" + user["id"],
        "subject_type": "user",
        "subject_user_id": user["id"],
        "name": user["name"],
        "credential_ids": [client["id"] for client in active],
        "knowledge_scopes": knowledge_scopes,
        "answer_scopes": [
            {"company_code": code, "domain_paths": list(dict.fromkeys(paths))}
            for code, paths in answer_scopes_by_company.items()
        ],
        "can_answer": any(client.get("can_answer", False) for client in active),
        "rate_limit_per_min": min(limits) if limits else 60,
    }


_rate_buckets: dict[str, list[float]] = {}


def _check_rate_limit(client: dict) -> None:
    limit = int(client.get("rate_limit_per_min", 60))
    if limit <= 0:
        return
    now = time.time()
    bucket = [t for t in _rate_buckets.get(client["id"], []) if now - t < 60]
    if len(bucket) >= limit:
        raise HTTPException(status_code=429, detail="超过每分钟调用上限")
    bucket.append(now)
    _rate_buckets[client["id"]] = bucket


def require_client(authorization: str = Header(default="")) -> dict:
    if not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="缺少 Bearer Token")
    key = authorization[len("Bearer "):].strip()
    internal_key = os.getenv("KNOWLEDGE_INTERNAL_API_KEY", "").strip()
    if internal_key and secrets.compare_digest(key, internal_key):
        return {
            "id": "fintech-backend",
            "knowledge_scopes": [
                {"company_code": code, "domain_paths": list(company["domains"])}
                for code, company in COMPANIES.items()
            ],
            "company_codes": list(COMPANIES),
            "domains": ["finance"],
            "can_answer": True,
            "rate_limit_per_min": 600,
        }
    user = auth_store.authenticate_user_api_key(key)
    if user:
        client = _effective_user_client(user)
        _check_rate_limit(client)
        return client
    client = _find_client_by_key(key)
    if not client:
        raise HTTPException(status_code=401, detail="凭证无效")
    if not client.get("enabled", True):
        raise HTTPException(status_code=401, detail="凭证已禁用")
    expires = client.get("expires_at")
    if expires:
        try:
            if datetime.fromisoformat(expires) < datetime.now(timezone.utc):
                raise HTTPException(status_code=401, detail="凭证已过期")
        except ValueError:
            pass
    _check_rate_limit(client)
    return client


def _client_domain_paths(client: dict, company_code: str) -> list[str]:
    """Return leaf-level access for new credentials and legacy credentials."""
    scopes = client.get("knowledge_scopes")
    if isinstance(scopes, list):
        paths = []
        for scope in scopes:
            if isinstance(scope, dict) and scope.get("company_code") == company_code:
                paths.extend(expand_domains(company_code, scope.get("domain_paths") or []))
        return list(dict.fromkeys(paths))

    allowed_companies = client.get("company_codes") or ["beijing_hangke"]
    if company_code not in allowed_companies:
        return []
    legacy_domains = client.get("domains") or []
    if not legacy_domains or "finance" in legacy_domains:
        return list(get_company(company_code)["domains"])
    return expand_domains(company_code, legacy_domains)


def _authorize_request_scope(client: dict, req: RetrieveRequest) -> None:
    if not get_company(req.company_code):
        raise HTTPException(status_code=422, detail="平台不存在")
    allowed = _client_domain_paths(client, req.company_code)
    if not allowed:
        raise HTTPException(status_code=403, detail="当前凭证无该平台知识域权限")
    if req.domain_paths:
        requested = expand_domains(req.company_code, req.domain_paths)
        if not requested:
            raise HTTPException(status_code=422, detail="存在不属于该平台的目录")
        if not set(requested).issubset(set(allowed)):
            raise HTTPException(status_code=403, detail="当前凭证无所请求目录的权限")
        req.domain_paths = requested
    else:
        req.domain_paths = allowed


def _normalize_knowledge_scopes(raw_scopes) -> list[dict]:
    if not isinstance(raw_scopes, list):
        raise HTTPException(status_code=422, detail="请选择知识域权限")
    normalized = []
    for raw in raw_scopes:
        if not isinstance(raw, dict) or raw.get("company_code") not in COMPANIES:
            raise HTTPException(status_code=422, detail="知识域权限包含无效平台")
        code = raw["company_code"]
        requested = raw.get("domain_paths")
        if not isinstance(requested, list) or not requested:
            continue
        invalid = [path for path in requested if not isinstance(path, str) or not expand_domains(code, [path])]
        if invalid:
            raise HTTPException(status_code=422, detail="知识域权限包含无效目录")
        leaves = expand_domains(code, requested)
        if leaves:
            normalized.append({"company_code": code, "domain_paths": leaves})
    if not normalized:
        raise HTTPException(status_code=422, detail="请至少选择一个三级目录权限")
    return normalized


# ---------- 查询路由规则 ----------

DEFAULT_ROUTING = {
    "rules": [
        {"id": "r-fund", "name": "基金业务", "keywords": ["基金设立", "基金运营", "清算注销"], "domains": ["基金层面"], "priority": 1, "enabled": True},
        {"id": "r-project", "name": "项目投资", "keywords": ["投前阶段", "投中阶段", "投后阶段"], "domains": ["项目层面"], "priority": 1, "enabled": True},
        {"id": "r-company", "name": "公司治理", "keywords": ["公司注册设立", "信息维护", "日常运营", "三会情况", "股东管理"], "domains": ["公司层面"], "priority": 1, "enabled": True},
    ],
    "default_domains": ["基金层面", "项目层面", "公司层面"],
    "llm_fallback": True,
}

# Character budgeting is intentionally conservative relative to model tokens.
# It replaces the old fixed contexts[:10] truncation while retaining a single
# global safety limit for every model call.
ANSWER_CONTEXT_CHAR_BUDGET = int(os.getenv("ANSWER_CONTEXT_CHAR_BUDGET", "320000"))


def _answer_context(candidates: list[dict]) -> list[str]:
    """Keep ordered evidence until one global budget is reached."""
    contexts: list[str] = []
    used = 0
    seen: set[str] = set()
    for candidate in candidates:
        src = candidate["source"]
        chunk_id = str(src.get("chunk_id") or src.get("chunk_index") or candidate.get("id"))
        if chunk_id in seen:
            continue
        seen.add(chunk_id)
        context = "[{0} / {1}] {2}".format(src.get("company_code", ""), src.get("domain_path", ""), src.get("content", ""))
        if contexts and used + len(context) > ANSWER_CONTEXT_CHAR_BUDGET:
            break
        contexts.append(context)
        used += len(context)
    return contexts


def _routing_store() -> dict:
    return _read_json(ROUTING_PATH, DEFAULT_ROUTING)


# ---------- 请求模型 ----------

class RetrieveRequest(BaseModel):
    query: str
    company_code: str = "beijing_hangke"
    domain_paths: Optional[list[str]] = None
    top_k: int = 8


class AnswerRequest(BaseModel):
    query: str
    company_code: str = "beijing_hangke"
    domain_paths: Optional[list[str]] = None
    top_k: int = 8


class CompleteListRequest(RetrieveRequest):
    page: int = 1
    page_size: int = 50


def _finance_filters(company_code: str, domain_paths: Optional[list[str]]) -> dict:
    if not get_company(company_code):
        raise HTTPException(status_code=422, detail="平台不存在")
    filters = {"company_code": company_code}
    invalid = [item for item in (domain_paths or []) if not expand_domains(company_code, [item])]
    if invalid:
        raise HTTPException(status_code=422, detail="存在不属于该平台的目录")
    if domain_paths:
        filters["domain_path"] = expand_domains(company_code, domain_paths)
    return filters


def _retrieve_finance(req, top_k: int):
    filters = _finance_filters(req.company_code, req.domain_paths)
    # One mixed-retrieval call across the allowed leaves, not up to 39
    # embedding/rerank requests for a first-level prefix selection.
    top, meta = retrieval.retrieve(es(), emb(), "finance", req.query, final_k=top_k, filters=filters)
    return top, [meta]


def _list_query_without_anchor(retrieval_meta: list[dict]) -> bool:
    """Do not ask the LLM to invent a complete list without a real section."""
    return bool(retrieval_meta) and all(
        item.get("list_query") and not item.get("section_anchor_found")
        for item in retrieval_meta
    )


def _table_rows(content: str) -> list[str]:
    """Read stored table rows directly instead of asking the LLM to recreate them."""
    marker = "【表格数据】"
    if marker not in content:
        return []
    rows = []
    for line in content.split(marker, 1)[1].splitlines():
        line = line.strip()
        if not line or re.fullmatch(r"\|?\s*[-:| ]+\|?", line):
            continue
        rows.append(line)
    return rows


def _source_evidence(candidate: dict, *, reference_index: int | None = None) -> dict:
    """Expose enough immutable structure for clients to audit a retrieval hit."""
    src = candidate["source"]
    result = {
        "collection": "finance",
        "title": src.get("title") or src.get("chunk_title") or "",
        "content": src.get("content", ""),
        "company_code": src.get("company_code"),
        "domain_path": src.get("domain_path"),
        "domain_level1": src.get("domain_level1"),
        "domain_level2": src.get("domain_level2"),
        "domain_level3": src.get("domain_level3"),
        "domain_leaf_id": src.get("domain_leaf_id"),
        "catalog_version": src.get("catalog_version"),
        "source": src.get("source", ""),
        "document_version": src.get("document_version", ""),
        "section_title": src.get("section_title", ""),
        "table_header": src.get("table_header", ""),
        "table_index": int(src.get("table_index") or 0),
        "table_row_start": int(src.get("table_row_start") or 0),
        "table_row_end": int(src.get("table_row_end") or 0),
        "document_table_count": int(src.get("document_table_count") or 0),
        "document_table_row_count": int(src.get("document_table_row_count") or 0),
        "chunk_index": int(src.get("chunk_index") or 0),
        "score": round(float(candidate.get("rerank_score", candidate.get("rrf_score", 0.0))), 6),
    }
    if reference_index is not None:
        result["reference_index"] = reference_index
    return result


def _complete_list(req: CompleteListRequest) -> dict:
    """Return a source-faithful, paginated list for an exact table section."""
    candidates, retrieval_meta = _retrieve_finance(req, retrieval.FINANCE_PROFILE["section_expand_limit"])
    exact = [item for item in candidates if item.get("section_expanded") and item.get("exact_subject_match")]
    if not exact:
        return {
            "available": False,
            "message": "未找到与清单主题精确对应的结构化章节，不能保证完整性。",
            "retrieval": retrieval_meta,
            "rows": [],
        }
    first = exact[0]["source"]
    doc_id = first.get("doc_id")
    section = first.get("section_title")
    # Section expansion has a safety cap for LLM context.  A complete list is
    # different: it must read every source row in the exact section, then let
    # the UI paginate it.  Do not reuse the retrieval expansion cap here.
    section_response = es().search(
        index=retrieval.FINANCE_PROFILE["index"],
        size=10000,
        query={"bool": {"filter": [
            {"term": {"company_code": first.get("company_code")}},
            {"term": {"domain_path": first.get("domain_path")}},
            {"term": {"doc_id": doc_id}},
            {"term": {"section_title.keyword": section}},
        ]}},
        sort=[{"chunk_index": {"order": "asc"}}],
    )
    section_sources = [hit["_source"] for hit in section_response["hits"]["hits"]]
    rows: list[str] = []
    seen_chunks: set[str] = set()
    for src in section_sources:
        chunk_id = str(src.get("chunk_id") or src.get("chunk_index"))
        if chunk_id in seen_chunks:
            continue
        seen_chunks.add(chunk_id)
        rows.extend(_table_rows(str(src.get("content", ""))))
    page_size = max(1, min(200, req.page_size))
    total = len(rows)
    total_pages = max(1, (total + page_size - 1) // page_size)
    page = max(1, min(req.page, total_pages))
    start = (page - 1) * page_size
    return {
        "available": bool(rows),
        "message": "" if rows else "命中章节不包含可分页的结构化表格行。",
        "section_title": section,
        "table_header": first.get("table_header", ""),
        "source": first.get("source", ""),
        "document_version": first.get("document_version", ""),
        "total": total,
        "page": page,
        "page_size": page_size,
        "total_pages": total_pages,
        "rows": rows[start:start + page_size],
        "retrieval": retrieval_meta,
    }


# ---------- 对外 API ----------

@app.get("/health")
def health():
    return {"status": "ok", "app": "科技金融企业知识库", "es": "up" if es().ping() else "down"}


@app.get("/api/companies")
def list_companies():
    return {"companies": public_companies()}


@app.get("/api/companies/{company_code}/tree")
def company_tree(company_code: str):
    if not get_company(company_code):
        raise HTTPException(status_code=422, detail="平台不存在")
    return {"company_code": company_code, "catalog_version": 2, "tree": domain_tree(company_code)}


@app.get("/api/companies/{company_code}")
def company_detail(company_code: str):
    company = get_company(company_code)
    if not company:
        raise HTTPException(status_code=404, detail="平台不存在")
    return {"code": company_code, **company}


@app.post("/api/v1/retrieve")
def retrieve(req: RetrieveRequest, client: dict = Depends(require_client)):
    _authorize_request_scope(client, req)
    candidates, retrieval_meta = _retrieve_finance(req, req.top_k)
    sources = []
    for candidate in candidates[:req.top_k]:
        sources.append(_source_evidence(candidate, reference_index=len(sources) + 1))
    return {
        "request_id": "req_" + secrets.token_hex(8),
        "answer": None,
        "answer_mode": "retrieval",
        "routed_to": req.domain_paths or get_company(req.company_code)["domains"],
        "sources": sources,
        "retrieval": retrieval_meta,
        "timing": {"total_ms": 0},
    }


@app.post("/api/v1/answer")
def answer(req: AnswerRequest, client: dict = Depends(require_client)):
    if not client.get("can_answer", False):
        raise HTTPException(status_code=403, detail="凭证无回答能力")
    if client.get("subject_type") == "user":
        answer_client = {**client, "knowledge_scopes": client.get("answer_scopes", [])}
        _authorize_request_scope(answer_client, req)
    else:
        _authorize_request_scope(client, req)
    # Answering may need the complete matched chapter.  Evidence returned to
    # clients remains short, while _answer_context applies the global budget.
    candidates, retrieval_meta = _retrieve_finance(req, retrieval.FINANCE_PROFILE["section_expand_limit"])
    contexts = []
    sources = []
    for candidate in candidates[:req.top_k]:
        src = candidate["source"]
        sources.append(_source_evidence(candidate))
    contexts = _answer_context(candidates)
    if not contexts:
        return {"request_id": "req_" + secrets.token_hex(8), "answer": "未检索到相关知识", "answer_mode": "retrieval_fallback", "sources": [], "retrieval": retrieval_meta}
    if _list_query_without_anchor(retrieval_meta):
        return {
            "request_id": "req_" + secrets.token_hex(8),
            "answer": "当前资料未找到可完整列举的对应章节，以下仅展示可能相关的原文证据；请调整关键词或补充资料。",
            "answer_mode": "retrieval_fallback", "sources": sources, "retrieval": retrieval_meta,
        }
    system = "你是科技金融企业知识库问答助手。只能依据给定资料回答，不得编造；资料不足时明确说明。回答使用简洁中文纯文本。"
    user = "资料：\n" + "\n\n".join(contexts) + "\n\n问题：" + req.query
    text = llm.chat(system, user)
    return {"request_id": "req_" + secrets.token_hex(8), "answer": text, "answer_mode": "answer", "sources": sources, "retrieval": retrieval_meta}


@app.post("/api/v1/list")
def complete_list(req: CompleteListRequest, client: dict = Depends(require_client)):
    _authorize_request_scope(client, req)
    return _complete_list(req)


# ---------- 页面内部检索 ----------

@app.post("/api/list")
def console_complete_list(body: dict):
    return _complete_list(CompleteListRequest(**body))

@app.post("/api/search")
def console_search(body: dict):
    """Local management-console search; the service itself is loopback-bound."""
    query = str(body.get("query", "")).strip()
    if not query:
        raise HTTPException(status_code=422, detail="请输入检索问题")
    domain_paths = body.get("domain_paths") or []
    company_code = body.get("company_code", "beijing_hangke")
    top_k = max(1, min(20, int(body.get("top_k", 8))))
    started = time.perf_counter()
    sources = []
    contexts = []
    req = RetrieveRequest(query=query, company_code=company_code, domain_paths=domain_paths or None, top_k=top_k)
    candidates, retrieval_meta = _retrieve_finance(
        req,
        retrieval.FINANCE_PROFILE["section_expand_limit"] if body.get("mode", "retrieve") == "answer" else top_k,
    )
    for candidate in candidates[:top_k]:
        src = candidate["source"]
        content = src.get("content", "")
        title = src.get("title") or src.get("chunk_title") or src.get("source") or "未命名知识"
        evidence = _source_evidence(candidate)
        evidence["title"] = title
        sources.append(evidence)
    contexts = _answer_context(candidates)
    mode = body.get("mode", "retrieve")
    generated_answer = None
    list_without_anchor = _list_query_without_anchor(retrieval_meta)
    if mode == "answer" and contexts and not list_without_anchor:
        system = "你是科技金融企业知识助手。只根据资料回答；资料不足时明确说明，不得编造。输出简洁纯文本。"
        prompt = "资料：\n" + "\n\n".join(contexts) + "\n\n问题：" + query
        generated_answer = llm.chat(system, prompt)
    return {
        "query": query,
        "answer": generated_answer or ("当前资料未找到可完整列举的对应章节，以下仅展示可能相关的原文证据；请调整关键词或补充资料。" if list_without_anchor else None),
        "answer_mode": "answer" if generated_answer else ("retrieval_fallback" if list_without_anchor else "retrieval"),
        "routed_to": domain_paths or get_company(company_code)["domains"],
        "sources": sources,
        "retrieval": retrieval_meta,
        "timing": {"total_ms": round((time.perf_counter() - started) * 1000)},
    }


# ---------- 文档中心 ----------

@app.get("/api/documents")
def console_documents(company_code: str = Query("beijing_hangke"), domain_paths: Optional[list[str]] = Query(None)):
    filters = _finance_filters(company_code, domain_paths)
    results = []
    chunks = scan(es(), index="kb_finance_doc", query={"query": {"bool": {"filter": retrieval._filters(filters)}}})
    grouped = {}
    for hit in chunks:
        src = hit["_source"]
        key = src.get("doc_id") or src.get("source") or hit["_id"]
        entry = grouped.setdefault(key, {
            "id": src.get("doc_id", hit["_id"]),
            "name": src.get("title") or Path(str(key)).name,
            "source": src.get("source", ""),
            "collection": "finance",
            "units": 0,
            "company_code": src.get("company_code", ""),
            "domain_path": src.get("domain_path", ""),
            **{field: src.get(field) for field in ("domain_level1", "domain_level2", "domain_level3", "domain_leaf_id", "catalog_version")},
            "updated_at": src.get("ingested_at", ""),
        })
        entry["units"] += 1
        if src.get("ingested_at", "") > entry["updated_at"]:
            entry["updated_at"] = src["ingested_at"]
    results.extend(grouped.values())
    results.sort(key=lambda item: item.get("updated_at", ""), reverse=True)
    return {"documents": results, "total": len(results)}


@app.delete("/api/documents/{document_id}")
def console_delete_document(document_id: str, company_code: str = Query("beijing_hangke")):
    _finance_filters(company_code, None)
    if not document_id or len(document_id) > 128:
        raise HTTPException(status_code=400, detail="文档编号无效")
    document = es().search(
        index="kb_finance_doc",
        size=1,
        query={"bool": {"filter": [
            {"term": {"doc_id": document_id}},
            {"term": {"company_code": company_code}},
        ]}},
    )
    hits = document["hits"]["hits"]
    if not hits:
        raise HTTPException(status_code=404, detail="文档不存在或已删除")
    source = str(hits[0]["_source"].get("source", ""))
    result = es().delete_by_query(
        index="kb_finance_doc",
        query={"bool": {"filter": [
            {"term": {"doc_id": document_id}},
            {"term": {"company_code": company_code}},
        ]}},
        refresh=True,
        conflicts="proceed",
    )
    deleted = int(result.get("deleted", 0))
    remaining = es().count(
        index="kb_finance_doc",
        query={"bool": {"filter": [
            {"term": {"company_code": company_code}},
            {"term": {"source": source}},
        ]}},
    )["count"]
    raw_file_deleted = False
    raw_file = (DATA_DIR / source).resolve()
    finance_root = (DATA_DIR / "finance").resolve()
    if not remaining and source and finance_root in raw_file.parents and raw_file.is_file():
        raw_file.unlink()
        raw_file_deleted = True
    _append_document_audit("deleted", document_id, "{0}:{1}".format(company_code, source))
    return {"document_id": document_id, "deleted_chunks": deleted, "raw_file_deleted": raw_file_deleted}


@app.get("/api/documents/{document_id}/detail")
def console_document_detail(
    document_id: str,
    company_code: str = Query("beijing_hangke"),
    query: str = Query("", max_length=200),
    page: int = Query(1, ge=1),
    page_size: int = Query(8, ge=1, le=50),
    full_text: bool = Query(False),
):
    """Return one document's original indexed chunks for the review screen."""
    _finance_filters(company_code, None)
    if not document_id or len(document_id) > 128:
        raise HTTPException(status_code=400, detail="文档编号无效")
    response = scan(es(),
        index="kb_finance_doc",
        query={"query": {"bool": {"filter": [
            {"term": {"doc_id": document_id}},
            {"term": {"company_code": company_code}},
        ]}}},
    )
    chunks = sorted((hit["_source"] for hit in response), key=lambda chunk: chunk.get("chunk_index", -1))
    if not chunks:
        raise HTTPException(status_code=404, detail="文档不存在或已删除")
    needle = query.strip().casefold()
    matched = [chunk for chunk in chunks if not needle or needle in str(chunk.get("content", "")).casefold()]
    total = len(matched)
    total_pages = max(1, (total + page_size - 1) // page_size)
    page = min(page, total_pages)
    start = (page - 1) * page_size
    page_chunks = matched[start:start + page_size]
    first = chunks[0]
    indices = [int(chunk.get("chunk_index", -1)) for chunk in chunks]
    integrity_ok = indices == list(range(len(chunks)))
    return {
        "document": {
            "id": document_id,
            "title": first.get("title") or Path(str(first.get("source", ""))).stem,
            "source": first.get("source", ""),
            "document_type": first.get("document_type", "unknown"),
            "parser": first.get("parser", ""),
            "company_code": first.get("company_code", ""),
            "domain_path": first.get("domain_path", ""),
            **{field: first.get(field) for field in ("domain_level1", "domain_level2", "domain_level3", "domain_leaf_id", "catalog_version")},
            "ingested_at": first.get("ingested_at", ""),
            "acl": first.get("acl", "all"),
            "document_version": first.get("document_version", ""),
        },
        "integrity": {"ok": integrity_ok, "chunk_total": len(chunks), "character_total": sum(len(str(chunk.get("content", ""))) for chunk in chunks)},
        "query": query.strip(),
        "matched_total": total,
        "page": page,
        "page_size": page_size,
        "total_pages": total_pages,
        "full_text": "\n\n".join(str(chunk.get("content", "")) for chunk in chunks) if full_text else "",
        "chunks": [{
            "index": int(chunk.get("chunk_index", 0)),
            "title": chunk.get("chunk_title", ""),
            "content": chunk.get("content", ""),
            "character_count": len(str(chunk.get("content", ""))),
        } for chunk in page_chunks],
    }


# ---------- 知识入库 ----------

ALLOWED_UPLOAD_SUFFIXES = {"finance": {".txt", ".md", ".pdf", ".docx"}}


def _safe_upload_name(name: str) -> str:
    candidate = Path(name or "").name.strip()
    if not candidate or candidate in {".", ".."} or candidate.startswith("._"):
        raise HTTPException(status_code=400, detail="文件名无效")
    return candidate


def _job_path(job_id: str) -> Path:
    return INGESTION_JOBS_DIR / (job_id + ".json")


def _write_job(job_id: str, payload: dict) -> None:
    _write_json(_job_path(job_id), payload)


def _run_ingestion_job(job_id: str, company_code: str, domain_path: str, mode: str, filenames: list[str]) -> None:
    job = _read_json(_job_path(job_id), {})
    job.update({"status": "processing", "progress": 25, "message": "正在解析文档并生成检索索引"})
    _write_job(job_id, job)
    leaf = get_leaf(company_code, domain_path)
    command = [sys.executable, "-m", "scripts.ingest_finance", "--files"] + [str(DATA_DIR / "finance" / company_code / leaf["leaf_id"] / name) for name in filenames]
    environment = os.environ.copy()
    environment["FINANCE_KB_INGESTION_MODE"] = mode
    environment["FINANCE_KB_COMPANY_CODE"] = company_code
    environment["FINANCE_KB_DOMAIN_PATH"] = domain_path
    try:
        completed = subprocess.run(
            command,
            cwd=str(PROJECT_DIR),
            env=environment,
            capture_output=True,
            text=True,
            timeout=1800,
            check=False,
        )
        if completed.returncode != 0:
            message = (completed.stderr or completed.stdout or "入库进程执行失败").strip()[-1200:]
            raise RuntimeError(message)
        job.update({
            "status": "succeeded",
            "progress": 100,
            "message": "文档已完成入库并刷新索引",
            "finished_at": datetime.now(timezone.utc).isoformat(),
            "output": completed.stdout.strip()[-1200:],
        })
    except Exception as exc:  # noqa: BLE001
        job.update({
            "status": "failed",
            "progress": 100,
            "message": "入库失败：{0}".format(exc),
            "finished_at": datetime.now(timezone.utc).isoformat(),
        })
    _write_job(job_id, job)


@app.post("/api/ingestion/upload", status_code=202)
async def console_ingestion_upload(
    background_tasks: BackgroundTasks,
    company_code: str = Form(...),
    domain_path: str = Form(...),
    mode: str = Form("standard"),
    files: list[UploadFile] = File(...),
):
    if not valid_domain(company_code, domain_path):
        raise HTTPException(status_code=422, detail="请选择当前平台的完整三级叶子目录")
    if mode not in {"standard", "semantic"}:
        raise HTTPException(status_code=422, detail="未知入库方式")
    if not files:
        raise HTTPException(status_code=422, detail="请选择文档")
    destination = (DATA_DIR / "finance" / company_code / get_leaf(company_code, domain_path)["leaf_id"]).resolve()
    destination.mkdir(parents=True, exist_ok=True)
    saved = []
    for upload in files:
        name = _safe_upload_name(upload.filename or "")
        if Path(name).suffix.lower() not in {".txt", ".md", ".pdf", ".docx"}:
            raise HTTPException(status_code=400, detail="{0} 不支持该知识域".format(name))
        target = (destination / name).resolve()
        if destination not in target.parents:
            raise HTTPException(status_code=400, detail="文件路径越界")
        size = 0
        with target.open("wb") as handle:
            while chunk := await upload.read(1024 * 1024):
                size += len(chunk)
                if size > 100 * 1024 * 1024:
                    handle.close()
                    target.unlink(missing_ok=True)
                    raise HTTPException(status_code=413, detail="单个文件不能超过 100MB")
                handle.write(chunk)
        await upload.close()
        saved.append(name)
    job_id = uuid.uuid4().hex
    job = {
        "id": job_id,
        "company_code": company_code,
        "domain_path": domain_path,
        **domain_metadata(company_code, domain_path),
        "mode": mode,
        "files": saved,
        "status": "queued",
        "progress": 8,
        "message": "文件已安全保存，等待入库处理",
        "created_at": datetime.now(timezone.utc).isoformat(),
    }
    _write_job(job_id, job)
    background_tasks.add_task(_run_ingestion_job, job_id, company_code, domain_path, mode, saved)
    return job


@app.get("/api/ingestion/jobs/{job_id}")
def console_ingestion_job(job_id: str):
    if not job_id.isalnum():
        raise HTTPException(status_code=400, detail="任务编号无效")
    job = _read_json(_job_path(job_id), None)
    if not job:
        raise HTTPException(status_code=404, detail="任务不存在")
    return job


# ---------- 本地用户管理 ----------

@app.get("/api/users")
def console_users():
    return {"users": auth_store.list_users()}


@app.post("/api/users", status_code=201)
def console_create_user(body: dict):
    credential_ids = _validated_credential_ids(body.get("credential_ids", []))
    try:
        user = auth_store.create_user(
            body.get("name", ""),
            body.get("position", ""),
            body.get("username", ""),
            body.get("password", ""),
            role=body.get("role", "user"),
            domains=body.get("domains", ["基金层面", "项目层面", "公司层面"]),
        )
        user = auth_store.set_user_credentials(user["id"], credential_ids)
        user, key = auth_store.rotate_user_api_key(user["id"])
        return {**user, "api_key": key}
    except ValueError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc


@app.put("/api/users/{user_id}")
def console_update_user(user_id: str, body: dict):
    current = auth_store.get_user(user_id)
    if not current:
        raise HTTPException(status_code=404, detail="用户不存在")
    credential_ids = _validated_credential_ids(body.get("credential_ids", current.get("credential_ids", [])))
    try:
        auth_store.update_user(
            user_id,
            name=body.get("name", current["name"]),
            position=body.get("position", current["position"]),
            username=body.get("username", current["username"]),
            role=body.get("role", current["role"]),
            domains=body.get("domains", current["domains"]),
            password=body.get("password") or None,
        )
        return auth_store.set_user_credentials(user_id, credential_ids)
    except ValueError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc


@app.post("/api/users/{user_id}/api-key")
def console_rotate_user_api_key(user_id: str):
    try:
        user, key = auth_store.rotate_user_api_key(user_id)
        return {"user": user, "api_key": key}
    except KeyError as exc:
        raise HTTPException(status_code=404, detail="用户不存在") from exc


@app.delete("/api/users/{user_id}")
def console_delete_user(user_id: str):
    try:
        auth_store.delete_user(user_id)
    except KeyError as exc:
        raise HTTPException(status_code=404, detail="用户不存在") from exc
    except ValueError as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc
    return {"deleted": user_id}


# ---------- 知识总览 API ----------

@app.get("/api/overview")
def overview(company_code: str = Query("beijing_hangke")):
    _finance_filters(company_code, None)
    chunk_count = es().count(index="kb_finance_doc", query={"term": {"company_code": company_code}})["count"]
    docs = console_documents(company_code, None)["documents"]
    documents = [{**item, "title": item["name"], "chunk_count": item["units"]} for item in docs[:50]]
    return {"document_count": len(docs), "chunk_count": chunk_count, "finance_docs": documents, "company_code": company_code, "company_name": get_company(company_code)["name"]}


# ---------- 路由规则 API ----------

@app.get("/api/routing/rules")
def get_routing_rules():
    return _routing_store()


@app.put("/api/routing/rules")
def put_routing_rules(store: dict):
    if not isinstance(store.get("rules"), list):
        raise HTTPException(status_code=422, detail="rules 必须是数组")
    _write_json(ROUTING_PATH, store)
    return _routing_store()


@app.post("/api/routing/rules")
def create_rule(rule: dict):
    store = _routing_store()
    rule = dict(rule)
    rule.setdefault("id", "r-" + secrets.token_hex(4))
    rule.setdefault("enabled", True)
    rule.setdefault("priority", 1)
    store["rules"].append(rule)
    _write_json(ROUTING_PATH, store)
    return rule


@app.delete("/api/routing/rules/{rule_id}")
def delete_rule(rule_id: str):
    store = _routing_store()
    store["rules"] = [r for r in store["rules"] if r.get("id") != rule_id]
    _write_json(ROUTING_PATH, store)
    return {"deleted": rule_id}


# ---------- 模型配置 API ----------

@app.get("/api/models")
def list_models():
    return {kind: model_config.masked_config(kind) for kind in ("llm", "embedding", "rerank")}


@app.put("/api/models/{kind}")
def update_model(kind: str, values: dict):
    if kind not in {"llm", "embedding", "rerank"}:
        raise HTTPException(status_code=404, detail="未知模型类型")
    allowed = {
        "llm": {"base_url", "api_key", "model"},
        "embedding": {"base_url", "api_key", "model", "dimension"},
        "rerank": {"base_url", "api_key", "model"},
    }[kind]
    cleaned = {k: v for k, v in values.items() if k in allowed}
    model_config.save_config(kind, cleaned)
    return {"kind": kind, "config": model_config.masked_config(kind)}


@app.post("/api/models/{kind}/test")
def test_model_connection(kind: str, values: dict):
    """Perform a minimal real request without persisting the submitted values."""
    if kind not in {"llm", "embedding", "rerank"}:
        raise HTTPException(status_code=404, detail="未知模型类型")
    config = dict(model_config.get_config(kind))
    for field in ("base_url", "api_key", "model", "dimension"):
        value = values.get(field)
        if isinstance(value, str) and value.strip():
            config[field] = value.strip()
        elif field == "dimension" and value:
            config[field] = value
    if not str(config.get("base_url", "")).strip() or not str(config.get("api_key", "")).strip() or not str(config.get("model", "")).strip():
        raise HTTPException(status_code=400, detail="请先填写 API 地址、模型名称和 API Key")
    started = time.perf_counter()
    try:
        if kind == "llm":
            from openai import OpenAI
            client = OpenAI(api_key=config["api_key"], base_url=config["base_url"].rstrip("/"))
            response = client.chat.completions.create(
                model=config["model"],
                messages=[{"role": "user", "content": "仅回复 OK"}],
                max_tokens=8,
                temperature=0,
            )
            detail = (response.choices[0].message.content or "").strip() or "模型已响应"
        elif kind == "embedding":
            provider = CloudEmbeddingProvider(
                base_url=config["base_url"],
                api_key=config["api_key"],
                model=config["model"],
                dimension=int(config.get("dimension") or 1024),
            )
            detail = "已返回 {0} 维向量".format(len(provider.embed_text("科技金融知识库连通性测试")))
        else:
            endpoint = str(config["base_url"]).rstrip("/")
            provider = "aliyun-qwen3" if "/services/rerank/" in endpoint else "configured"
            scores, _ = rerank_api.cloud_rerank(
                {"provider": provider, "base_url": endpoint, "api_key": config["api_key"], "model": config["model"], "timeout": 20},
                "科技金融知识库测试",
                ["科技金融企业知识管理", "无关测试文本"],
            )
            detail = "已返回 {0} 条重排结果".format(len(scores))
    except Exception as error:  # noqa: BLE001
        raise HTTPException(status_code=400, detail="连通性测试失败：{0}".format(str(error)[:500])) from error
    return {"kind": kind, "ok": True, "latency_ms": round((time.perf_counter() - started) * 1000), "message": "连通成功：{0}".format(detail[:160])}


# ---------- 机器凭证 API ----------

@app.get("/api/credentials")
def list_credentials():
    return {"clients": [{k: v for k, v in c.items() if k != "api_key_sha256"} for c in _credential_store()["clients"]]}


@app.post("/api/credentials")
def create_credential(body: dict):
    store = _credential_store()
    key = _new_api_key()
    if "knowledge_scopes" in body:
        knowledge_scopes = _normalize_knowledge_scopes(body.get("knowledge_scopes"))
    else:
        # Backward-compatible creation for callers using the former company_codes/domains fields.
        company_codes = [code for code in body.get("company_codes", []) if code in COMPANIES] or ["beijing_hangke"]
        domains = body.get("domains") or ["基金层面", "项目层面", "公司层面"]
        knowledge_scopes = _normalize_knowledge_scopes([
            {"company_code": code, "domain_paths": domains} for code in company_codes
        ])
    company_codes = [scope["company_code"] for scope in knowledge_scopes]
    selected_leaves = [path for scope in knowledge_scopes for path in scope["domain_paths"]]
    client = {
        "id": "c-" + secrets.token_hex(4),
        "name": str(body.get("name") or "未命名").strip()[:80],
        "api_key_sha256": _sha256(key),
        "knowledge_scopes": knowledge_scopes,
        "company_codes": company_codes,
        "domains": list(dict.fromkeys(path.split("/", 1)[0] for path in selected_leaves)),
        "can_answer": body.get("can_answer", False),
        "rate_limit_per_min": max(1, min(600, int(body.get("rate_limit_per_min", 60)))),
        "expires_at": body.get("expires_at"),
        "enabled": True,
        "created_at": datetime.now(timezone.utc).isoformat(),
    }
    store["clients"].append(client)
    _save_credentials(store)
    return {**{k: v for k, v in client.items() if k != "api_key_sha256"}, "api_key": key}


@app.delete("/api/credentials/{client_id}")
def delete_credential(client_id: str):
    store = _credential_store()
    store["clients"] = [c for c in store["clients"] if c.get("id") != client_id]
    _save_credentials(store)
    auth_store.remove_credential_bindings(client_id)
    return {"deleted": client_id}


# ---------- 前端静态 ----------

@app.get("/")
def index():
    index_path = STATIC_DIR / "index.html"
    if index_path.exists():
        return FileResponse(index_path)
    return {"message": "科技金融企业知识库系统 API 已就绪"}


if STATIC_DIR.exists():
    app.mount("/static", StaticFiles(directory=str(STATIC_DIR)), name="static")
