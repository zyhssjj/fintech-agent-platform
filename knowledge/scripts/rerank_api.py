"""HTTP adapters for supported cloud reranking protocols."""

import requests


DASHSCOPE_NATIVE_PATH = "/api/v1/services/rerank/text-rerank/text-rerank"


class RerankAPIError(RuntimeError):
    """Raised when a reranking service returns an invalid response."""


def rerank_endpoint(model_config):
    base_url = model_config["base_url"].rstrip("/")
    if model_config["provider"] == "aliyun-qwen3" or "/services/rerank/" in base_url:
        # DashScope native reranking URLs are complete endpoints. Appending
        # /reranks would turn the official URL into a non-existent route.
        return base_url
    return base_url if base_url.endswith("/reranks") else base_url + "/reranks"


def rerank_request_body(model_config, question, documents):
    if model_config["provider"] == "aliyun-qwen3":
        return {
            "model": model_config["model"],
            "input": {
                "query": question,
                "documents": documents,
            },
            "parameters": {
                # The knowledge base already owns the source text, so the API
                # only needs to return indexes and scores.
                "return_documents": False,
                "top_n": len(documents),
            },
        }
    return {
        "model": model_config["model"],
        "query": question,
        "documents": documents,
        "top_n": len(documents),
    }


def response_error(response):
    try:
        payload = response.json()
    except (TypeError, ValueError):
        payload = {}
    code = str(payload.get("code") or payload.get("error", {}).get("code") or "").strip()
    message = str(
        payload.get("message")
        or payload.get("error", {}).get("message")
        or getattr(response, "text", "")
        or "服务未返回错误详情"
    ).strip()
    details = " · ".join(item for item in (code, message[:400]) if item)
    return "重排API返回 HTTP {0}{1}".format(
        response.status_code,
        ": " + details if details else "",
    )


def cloud_rerank(model_config, question, documents):
    response = requests.post(
        rerank_endpoint(model_config),
        headers={
            "Authorization": "Bearer {0}".format(model_config["api_key"]),
            "Content-Type": "application/json",
        },
        json=rerank_request_body(model_config, question, documents),
        timeout=model_config["timeout"],
    )
    if not 200 <= response.status_code < 300:
        raise RerankAPIError(response_error(response))
    try:
        payload = response.json()
    except (TypeError, ValueError) as error:
        raise RerankAPIError("重排API没有返回有效JSON") from error
    results = payload.get("results")
    if results is None:
        results = payload.get("output", {}).get("results")
    if not isinstance(results, list) or not results:
        raise RerankAPIError("重排API没有返回有效排序结果")
    scores = {}
    for item in results:
        try:
            index = int(item["index"])
            score = float(item["relevance_score"])
        except (KeyError, TypeError, ValueError) as error:
            raise RerankAPIError("重排API返回了无效排序数据") from error
        if not 0 <= index < len(documents):
            raise RerankAPIError("重排API返回了无效候选编号")
        scores[index] = score
    if len(scores) != len(documents):
        raise RerankAPIError("重排API没有覆盖全部候选")
    usage = payload.get("usage") or payload.get("output", {}).get("usage") or {}
    return scores, usage.get("total_tokens")
