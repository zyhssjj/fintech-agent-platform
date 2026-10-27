"""Chapter-aware hybrid retrieval for long and semi-structured enterprise documents."""

from __future__ import annotations

import re

from scripts import rerank_api


FINANCE_PROFILE = {
    "index": "kb_finance_doc",
    "bm25_fields": ["content^2", "section_title^6", "chunk_title^4", "table_header^3", "title^2", "keywords^3"],
    "vector_field": "content_vector",
    "bm25_k": 120,
    "vector_k": 120,
    "final_k": 120,
    # Expansion is governed by the answer-context budget, not an arbitrary
    # small chunk count.  Keep only a generous transport guard for ES.
    "section_expand_limit": 10_000,
}

LIST_QUERY_RE = re.compile(r"(?:多少(?:个|项|条|类)?|有哪些|有多少(?:个|项|条|类)?|全部列出|请列出|列出所有|所有[^，。？?]*|逐项|完整清单|包含哪些)")
SUBJECT_PATTERNS = (
    re.compile(r"(?:多少(?:个|项|条|类)?|有哪些|有多少(?:个|项|条|类)?|全部列出|逐项|完整清单)\s*([^？?，。,；;：:]{2,40})"),
    re.compile(r"([^？?，。,；;：:]{2,40})\s*(?:有哪些|有多少(?:个|项|条|类)?|多少(?:个|项|条|类)?)"),
    re.compile(r"(?:记录|包含|列出)\s*([^？?，。,；;：:]{2,40})"),
)
# These terms frequently occur in questions but do not identify a document
# section.  Keeping them out of the anchor matcher prevents broad questions
# from expanding an arbitrary "系统" or "说明" chapter.
SECTION_QUERY_STOPWORDS = {
    "哪些", "多少", "什么", "怎么", "如何", "请问", "请", "所有", "全部", "相关",
    "内容", "情况", "信息", "文档", "资料", "系统", "功能", "说明", "介绍", "可以",
    "需要", "是否", "里面", "其中", "关于", "这个", "那个", "一下", "一下子",
}


def rerank_config() -> dict:
    from scripts import model_config

    config = model_config.get_config("rerank")
    return {"provider": "configured", "base_url": config.get("base_url", ""), "api_key": config.get("api_key", ""), "model": config.get("model", ""), "timeout": 30}


def extract_query_subject(query: str) -> str:
    """Extract the requested object so exact chapter matches are not buried by prose."""
    compact = re.sub(r"\s+", " ", query).strip()
    for pattern in SUBJECT_PATTERNS:
        match = pattern.search(compact)
        if not match:
            continue
        subject = match.group(1).strip(" ：:，,。.!！？?")
        subject = re.sub(r"^(?:系统中|文档中|其中|记录了|包含)", "", subject).strip()
        if 2 <= len(subject) <= 40:
            return subject
    return ""


def is_list_query(query: str) -> bool:
    return bool(LIST_QUERY_RE.search(query))


def _filters(filters: dict | None) -> list[dict]:
    return [{"terms" if isinstance(value, list) else "term": {key: value}} for key, value in (filters or {}).items() if value]


def _bm25_search(es, query: str, filters: dict | None) -> list[dict]:
    response = es.search(index=FINANCE_PROFILE["index"], size=FINANCE_PROFILE["bm25_k"], query={"bool": {"must": [{"multi_match": {"query": query, "fields": FINANCE_PROFILE["bm25_fields"], "type": "best_fields"}}], "filter": _filters(filters)}})
    return [{"id": hit["_id"], "source": hit["_source"], "score": hit["_score"]} for hit in response["hits"]["hits"]]


def _vector_search(es, query_vector: list[float], filters: dict | None) -> list[dict]:
    size = FINANCE_PROFILE["vector_k"]
    response = es.search(index=FINANCE_PROFILE["index"], size=size, knn={"field": FINANCE_PROFILE["vector_field"], "query_vector": query_vector, "k": size, "num_candidates": min(size * 4, 500), "filter": _filters(filters)})
    return [{"id": hit["_id"], "source": hit["_source"], "score": hit["_score"]} for hit in response["hits"]["hits"]]


def _section_title(source: dict) -> str:
    title = str(source.get("section_title") or source.get("chunk_title") or "").strip()
    if title:
        return title
    content = str(source.get("content", ""))
    match = re.search(r"【章节】\s*([^\n]+)", content)
    if match:
        return match.group(1).strip()
    match = re.search(r"^#{1,6}\s+(.+)$", content, flags=re.MULTILINE)
    return match.group(1).strip() if match else ""


def _subject_matches(source: dict, subject: str) -> bool:
    if not subject:
        return False
    fields = (_section_title(source), str(source.get("table_header", "")), str(source.get("chunk_title", "")))
    return subject.casefold() in " ".join(fields).casefold()


def _question_focus_terms(query: str) -> set[str]:
    """Extract the object around common Chinese question forms."""
    terms: set[str] = set()
    compact = re.sub(r"\s+", "", query)
    for match in re.finditer(r"什么([\u4e00-\u9fff]{2,12})", compact):
        terms.add(match.group(1))
    for match in re.finditer(r"([\u4e00-\u9fff]{2,12})(?:是什么|是怎样|如何)", compact):
        terms.add(match.group(1))
    return terms


def _query_section_match_score(source: dict, query: str) -> int:
    """Match meaningful words in a natural-language question to a heading.

    List questions have an explicit subject, but ordinary questions such as
    "系统定位说明了什么" do not.  Build small Chinese word windows from the
    question so the real heading phrase can still become an expansion anchor.
    """
    section = _section_title(source).casefold()
    if not section:
        return 0
    terms: set[str] = set()
    focus_terms = _question_focus_terms(query)
    for focus in focus_terms:
        if focus not in SECTION_QUERY_STOPWORDS and focus.casefold() in section:
            # A direct match to the object asked after "什么" is more reliable
            # than the document title mentioned in the same sentence.
            return len(focus) + 100
    for phrase in re.findall(r"[\u4e00-\u9fff]{2,}", query):
        if phrase not in SECTION_QUERY_STOPWORDS:
            terms.add(phrase)
        # Headings are commonly a short phrase embedded in a longer sentence.
        # Two-character entities (for example "智能体") are retained, while
        # generic terms are excluded through SECTION_QUERY_STOPWORDS.
        for length in range(2, min(12, len(phrase)) + 1):
            for start in range(0, len(phrase) - length + 1):
                term = phrase[start:start + length]
                if term not in SECTION_QUERY_STOPWORDS:
                    terms.add(term)
    matched = [len(term) for term in terms if term.casefold() in section]
    return max(matched, default=0)


def _merge_hits(bm25_hits: list[dict], vector_hits: list[dict], subject: str, query: str) -> list[dict]:
    """Reciprocal-rank fusion plus deterministic boosts for exact structural hits."""
    merged: dict[str, dict] = {}
    for channel, hits in (("bm25", bm25_hits), ("vector", vector_hits)):
        for rank, hit in enumerate(hits, start=1):
            item = merged.setdefault(hit["id"], {**hit, "rrf_score": 0.0, "retrieval_sources": []})
            item["rrf_score"] += 1.0 / (60 + rank)
            item["retrieval_sources"].append(channel)
    for item in merged.values():
        item["section_title"] = _section_title(item["source"])
        item["exact_subject_match"] = _subject_matches(item["source"], subject)
        item["query_section_match_score"] = _query_section_match_score(item["source"], query)
        item["query_section_match"] = item["query_section_match_score"] > 0
        # An extracted list subject is the strongest deterministic signal.  A
        # phrase found directly in the user's question is deliberately weaker.
        item["structure_boost"] = 0.12 if item["exact_subject_match"] else (0.06 if item["query_section_match"] else 0.0)
        item["hybrid_score"] = item["rrf_score"] + item["structure_boost"]
    return sorted(merged.values(), key=lambda item: item["hybrid_score"], reverse=True)


def _rerank(query: str, candidates: list[dict]) -> tuple[list[dict], bool]:
    if not candidates:
        return candidates, False
    documents = [item["source"].get("content", "") or item["source"].get("chunk_title", "") for item in candidates]
    try:
        scores, _ = rerank_api.cloud_rerank(rerank_config(), query, documents)
    except Exception as exc:  # noqa: BLE001
        print("  重排不可用，保留章节感知混合排序: {0}".format(exc))
        return candidates, False
    for index, candidate in enumerate(candidates):
        candidate["rerank_score"] = scores.get(index, 0.0)
        candidate["final_score"] = candidate["rerank_score"] + candidate["structure_boost"]
    candidates.sort(key=lambda item: item.get("final_score", item["hybrid_score"]), reverse=True)
    return candidates, True


def _expand_matching_sections(es, candidates: list[dict], filters: dict | None, list_query: bool) -> list[dict]:
    """Load the actual hit section in original order, never from document start."""
    anchors = [item for item in candidates if item.get("exact_subject_match") and item.get("section_title")]
    if not anchors:
        anchors = [item for item in candidates if item.get("query_section_match") and item.get("section_title")]
        anchors.sort(key=lambda item: (item.get("query_section_match_score", 0), item["hybrid_score"]), reverse=True)
    if not anchors:
        # A list/count request must never present a semantically similar
        # section as a complete catalogue.  Keep the ordinary evidence hits,
        # but do not fabricate a section expansion.
        if list_query:
            return []
        anchors = [item for item in candidates if item.get("section_title")][:1]
    if not anchors:
        return []
    expanded: list[dict] = []
    seen: set[str] = set()
    for anchor in anchors[:2]:
        source = anchor["source"]
        section = anchor["section_title"]
        query_filters = _filters(filters) + [
            {"term": {"doc_id": source.get("doc_id")}},
            # section_title is analysed for search, so exact section loading
            # must target its keyword subfield rather than analysed tokens.
            {"term": {"section_title.keyword": section}},
        ]
        response = es.search(index=FINANCE_PROFILE["index"], size=FINANCE_PROFILE["section_expand_limit"], query={"bool": {"filter": query_filters}}, sort=[{"chunk_index": {"order": "asc"}}])
        for hit in response["hits"]["hits"]:
            if hit["_id"] in seen:
                continue
            seen.add(hit["_id"])
            expanded.append({
                "id": hit["_id"], "source": hit["_source"], "rrf_score": anchor["rrf_score"],
                "section_title": section, "exact_subject_match": anchor.get("exact_subject_match", False),
                "query_section_match": anchor.get("query_section_match", False),
                "section_expanded": True, "structure_boost": anchor["structure_boost"],
            })
    return expanded


def retrieve(es, emb, domain: str, query: str, final_k: int | None = None, filters: dict | None = None) -> tuple[list[dict], dict]:
    if domain != "finance":
        raise ValueError("当前服务仅支持科技金融知识域")
    subject = extract_query_subject(query)
    list_query = is_list_query(query)
    bm25_hits = _bm25_search(es, query, filters)
    if subject and subject != query:
        bm25_hits = _bm25_search(es, subject, filters) + bm25_hits
    vector_hits = _vector_search(es, emb.embed_text(query), filters)
    merged, rerank_applied = _rerank(query, _merge_hits(bm25_hits, vector_hits, subject, query))
    expanded = _expand_matching_sections(es, merged, filters, list_query)
    expanded_ids = {item["id"] for item in expanded}
    combined = expanded + [item for item in merged if item["id"] not in expanded_ids]
    top = combined[: final_k or FINANCE_PROFILE["final_k"]]
    return top, {
        "domain": "finance", "query_subject": subject, "list_query": list_query,
        "exact_section_match_count": sum(1 for item in merged if item.get("exact_subject_match")),
        "section_anchor_found": bool(expanded), "bm25_count": len(bm25_hits),
        "vector_count": len(vector_hits), "union_count": len(merged),
        "section_expanded_count": len(expanded), "rerank_applied": rerank_applied,
        "returned": len(top),
    }
