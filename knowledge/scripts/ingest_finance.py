"""科技金融长文档入库：解析文档 → 语义增强切块 → 向量化 → 写入 kb_finance_doc。

用法：
    设置 FINANCE_KB_COMPANY_CODE 和 FINANCE_KB_DOMAIN_PATH 为合法 V2 平台与三级路径
    python -m scripts.ingest_finance --dry-run
    python -m scripts.ingest_finance
"""

from __future__ import annotations

import argparse
import hashlib
import os
import re
from datetime import datetime, timezone
from pathlib import Path

from dotenv import load_dotenv
from elasticsearch import Elasticsearch, helpers

from scripts import llm, semantic_chunking
from scripts.embeddings import configured_provider
from scripts.company_catalog import domain_metadata


SUPPORTED_SUFFIXES = {".md", ".txt", ".docx", ".pdf", ".doc"}
CHUNK_SIZE = 600
CHUNK_OVERLAP = 80


class DocumentReadError(RuntimeError):
    pass


def normalize_text(value: str) -> str:
    lines = [re.sub(r"\s+", " ", line).strip() for line in value.splitlines()]
    return "\n".join(line for line in lines if line)


def read_document(path: Path) -> tuple[str, str]:
    suffix = path.suffix.lower()
    if suffix in {".md", ".txt"}:
        return normalize_text(path.read_text(encoding="utf-8")), "text"
    if suffix == ".docx":
        import docx  # lazy
        from docx.oxml.table import CT_Tbl
        from docx.oxml.text.paragraph import CT_P
        from docx.table import Table
        from docx.text.paragraph import Paragraph

        doc = docx.Document(str(path))
        parts = []
        table_index = 0
        # doc.paragraphs and doc.tables are separate lists.  Combining them
        # loses their original order, which breaks the heading-to-table link
        # needed for trustworthy long-document retrieval.
        for child in doc.element.body.iterchildren():
            if isinstance(child, CT_P):
                value = Paragraph(child, doc).text.strip()
                if value:
                    parts.append(value)
            elif isinstance(child, CT_Tbl):
                table_index += 1
                parts.append("【表格】第{0}个数据表".format(table_index))
                for row in Table(child, doc).rows:
                    values = [cell.text.strip() for cell in row.cells]
                    if any(values):
                        parts.append(" | ".join(values))
        return normalize_text("\n".join(parts)), "docx"
    if suffix == ".pdf":
        from pypdf import PdfReader  # lazy（文字层）
        text = "\n".join((page.extract_text() or "") for page in PdfReader(str(path)).pages)
        return normalize_text(text), "pdf"
    if suffix == ".doc":
        raise DocumentReadError("legacy .doc 不支持，请转 .docx")
    raise DocumentReadError("不支持的扩展名: " + suffix)


def chunk_text(text: str, size: int, overlap: int) -> list[str]:
    """标准切块兜底：段落优先 + 超长段按句末标点再分。"""
    units = [line.strip() for line in text.splitlines() if line.strip()]
    chunks: list[str] = []
    current = ""
    for unit in units:
        if len(unit) > size:
            if current:
                chunks.append(current)
                current = ""
            start = 0
            while start < len(unit):
                end = min(start + size, len(unit))
                if end < len(unit):
                    boundary = max(unit.rfind(mark, start + size // 2, end) for mark in "。！？；;\n")
                    if boundary >= start + size // 2:
                        end = boundary + 1
                piece = unit[start:end].strip()
                if piece:
                    chunks.append(piece)
                start = end
            continue
        candidate = unit if not current else current + "\n" + unit
        if len(candidate) <= size:
            current = candidate
            continue
        chunks.append(current)
        current = current[-overlap:] + "\n" + unit if overlap else unit
    if current:
        chunks.append(current)
    return [chunk for chunk in chunks if len(chunk) >= 30]


def _heading(line: str) -> str:
    # Numbered table rows such as "1 | 产品名称 | ..." must never become a
    # section heading.  Treating them as headings resets the table accumulator
    # and silently discards nearly all rows from a long DOCX table.
    if _looks_like_table_row(line):
        return ""
    match = re.match(r"^(?:#{1,6}\s+|第[一二三四五六七八九十0-9]+[章节部分]\s*|[0-9]+(?:\.[0-9]+)*[.．、]?\s+)(.+)$", line.strip())
    return match.group(1).strip() if match else ""


def _looks_like_table_row(line: str) -> bool:
    # DOCX rows are serialized with " | ". A two-column table has exactly one
    # separator, and must not silently fall back to prose or disappear.
    return line.count("|") >= 1 or line.count("\t") >= 1


def structured_chunk(text: str, size: int, overlap: int) -> list[dict]:
    """Keep heading/table-header context on every independently retrievable chunk."""
    lines = [line.strip() for line in text.splitlines() if line.strip()]
    plans: list[dict] = []
    section = "文档正文"
    table_index = 0
    table_marker_pending = False
    table_header = ""
    paragraph_buffer: list[str] = []
    table_rows: list[str] = []

    def flush_paragraphs() -> None:
        nonlocal paragraph_buffer
        if not paragraph_buffer:
            return
        body = "\n".join(paragraph_buffer)
        for chunk in chunk_text(body, size, overlap):
            plans.append({"content": "【章节】{0}\n{1}".format(section, chunk), "title": section, "keywords": [section], "section_title": section, "table_header": "", "is_structured": False})
        paragraph_buffer = []

    def flush_table() -> None:
        nonlocal table_rows, table_header, table_marker_pending
        if not table_rows:
            return
        group: list[str] = []
        length = 0
        row_start = 1
        for row_number, row in enumerate(table_rows, start=1):
            estimated = len(row) + 1
            if group and length + estimated > size:
                plans.append({"content": "【章节】{0}\n【表格】第{1}个数据表\n【表头】{2}\n【表格数据】\n{3}".format(section, table_index, table_header, "\n".join(group)), "title": section, "keywords": [section, table_header], "section_title": section, "table_header": table_header, "table_index": table_index, "table_row_start": row_start, "table_row_end": row_number - 1, "is_structured": True})
                group, length = [], 0
                row_start = row_number
            group.append(row)
            length += estimated
        if group:
            plans.append({"content": "【章节】{0}\n【表格】第{1}个数据表\n【表头】{2}\n【表格数据】\n{3}".format(section, table_index, table_header, "\n".join(group)), "title": section, "keywords": [section, table_header], "section_title": section, "table_header": table_header, "table_index": table_index, "table_row_start": row_start, "table_row_end": len(table_rows), "is_structured": True})
        table_rows, table_header = [], ""
        table_marker_pending = False

    for line in lines:
        table_match = re.fullmatch(r"【表格】第(\d+)个数据表", line)
        if table_match:
            flush_paragraphs()
            flush_table()
            table_index = int(table_match.group(1))
            table_marker_pending = True
            continue
        title = _heading(line)
        if title:
            flush_paragraphs()
            flush_table()
            section = title
            continue
        if _looks_like_table_row(line):
            flush_paragraphs()
            if not table_header:
                # DOCX supplies an explicit table marker. Markdown/TXT tables do
                # not, so assign them a stable sequential table number locally.
                if not table_marker_pending:
                    table_index += 1
                table_header = line
            else:
                table_rows.append(line)
            continue
        flush_table()
        paragraph_buffer.append(line)
    flush_paragraphs()
    flush_table()
    return plans or [{"content": chunk, "title": "", "keywords": [], "section_title": "", "table_header": "", "table_index": 0, "table_row_start": 0, "table_row_end": 0, "is_structured": False} for chunk in chunk_text(text, size, overlap)]


def validate_structured_plans(plans: list[dict]) -> dict:
    """Prove that every structured table row is present once and in order."""
    tables: dict[int, list[dict]] = {}
    for plan in plans:
        if not plan.get("is_structured"):
            continue
        table_index = int(plan.get("table_index") or 0)
        if not table_index or not plan.get("section_title") or not plan.get("table_header"):
            raise DocumentReadError("结构化表格缺少章节、表头或表格编号")
        tables.setdefault(table_index, []).append(plan)
    total_rows = 0
    section_rows: dict[str, int] = {}
    for table_index, chunks in tables.items():
        chunks.sort(key=lambda item: int(item["table_row_start"]))
        expected_start = 1
        for chunk in chunks:
            start, end = int(chunk["table_row_start"]), int(chunk["table_row_end"])
            if start != expected_start or end < start:
                raise DocumentReadError("第{0}个数据表行序号不连续".format(table_index))
            expected_start = end + 1
        row_count = expected_start - 1
        total_rows += row_count
        section_title = str(chunks[0]["section_title"])
        section_rows[section_title] = section_rows.get(section_title, 0) + row_count
    declared_mismatches = []
    for section_title, row_count in section_rows.items():
        declared = re.search(r"共\s*(\d+)\s*项", section_title)
        if declared and int(declared.group(1)) != row_count:
            declared_mismatches.append({"section_title": section_title, "declared": int(declared.group(1)), "actual": row_count})
    if declared_mismatches:
        raise DocumentReadError("表格声明条目数与实际解析行数不一致: {0}".format(declared_mismatches[:3]))
    return {"table_count": len(tables), "table_row_count": total_rows}


def semantic_chunk(text: str, progress=None) -> tuple[list[dict], str]:
    """语义增强切块；失败回退标准切块。返回 (plans, mode)。"""
    if os.getenv("FINANCE_KB_INGESTION_MODE", "semantic").strip().lower() == "standard":
        return structured_chunk(text, CHUNK_SIZE, CHUNK_OVERLAP), "structured"
    try:
        result = semantic_chunking.plan_semantic_chunks(
            text,
            "finance",
            call_model=lambda system, user: llm.chat(system, user, json_mode=True),
            progress=progress,
        )
        plans = result["plans"]
        # Semantic planning is accepted only for prose; structural table chunks stay immutable.
        if any(_looks_like_table_row(line) for line in text.splitlines()):
            return structured_chunk(text, CHUNK_SIZE, CHUNK_OVERLAP), "structured"
        for plan in plans:
            plan.setdefault("section_title", plan.get("title", ""))
            plan.setdefault("table_header", "")
            plan.setdefault("is_structured", False)
        return plans, "semantic"
    except (semantic_chunking.SemanticChunkingError, llm.LLMError) as exc:
        print("  语义切块失败，回退标准切块: {0}".format(exc))
        return structured_chunk(text, CHUNK_SIZE, CHUNK_OVERLAP), "structured-fallback"


def stable_doc_id(source: str) -> str:
    return hashlib.sha256(source.encode("utf-8")).hexdigest()[:24]


def content_hash(text: str) -> str:
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def document_type(path: Path) -> str:
    parts = path.resolve().relative_to((Path("data") / "finance").resolve()).parts[:-1]
    return parts[0] if parts else "general"


def ensure_finance_index(es: Elasticsearch) -> None:
    from scripts.create_indices import INDEX_DEFINITIONS, index_definition
    if not es.indices.exists(index="kb_finance_doc"):
        es.indices.create(index="kb_finance_doc", **index_definition(INDEX_DEFINITIONS["kb_finance_doc"]))
        print("created kb_finance_doc")


def main() -> None:
    load_dotenv()
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--data-root", default="data/finance")
    parser.add_argument("--files", nargs="+", help="仅处理指定文件，供异步上传任务使用")
    parser.add_argument("--dry-run", action="store_true", help="只解析 + 切块，不写 ES")
    args = parser.parse_args()
    company_code = os.getenv("FINANCE_KB_COMPANY_CODE", "beijing_hangke")
    domain_path = os.getenv("FINANCE_KB_DOMAIN_PATH", "")
    metadata = domain_metadata(company_code, domain_path)

    files = [Path(item) for item in args.files] if args.files else []
    if not files:
        for path in sorted(Path(args.data_root).rglob("*")):
            if not path.is_file() or path.name.startswith("._") or path.name == ".gitkeep":
                continue
            if path.suffix.lower() in SUPPORTED_SUFFIXES:
                files.append(path)

    for path in files:
        if not path.is_file() or path.suffix.lower() not in SUPPORTED_SUFFIXES:
            raise DocumentReadError("不存在或不支持的文件: " + str(path))

    if not files:
        print("未找到科技金融文档（{0} 下无受支持文件）".format(args.data_root))
        return
    print("找到 {0} 个科技金融文档".format(len(files)))

    emb = configured_provider()
    es = None
    if not args.dry_run:
        es = Elasticsearch(os.getenv("ES_URL", "http://localhost:9202"))
        if not es.ping():
            raise RuntimeError("无法连接 Elasticsearch（{0}）".format(os.getenv("ES_URL", "http://localhost:9202")))
        ensure_finance_index(es)

    for path in files:
        source = path.resolve().relative_to(Path(args.data_root).resolve().parent).as_posix()
        try:
            text, parser_name = read_document(path)
            if not text.strip():
                raise DocumentReadError("文档无可用文本")
        except Exception as exc:  # noqa: BLE001
            raise DocumentReadError("无法解析 {0}: {1}".format(source, exc)) from exc

        print("\n=== {0}（{1} 字符）===".format(source, len(text)))
        plans, mode = semantic_chunk(text, progress=lambda stage, msg: print("  [{0}] {1}".format(stage, msg)))
        integrity = validate_structured_plans(plans)
        print("  切块方式: {0}，块数: {1}".format(mode, len(plans)))
        print("  结构校验: {0} 个数据表，{1} 条完整数据行".format(integrity["table_count"], integrity["table_row_count"]))

        if args.dry_run:
            for index, plan in enumerate(plans):
                print("  - chunk{0}（{1}字）: {2}".format(index, len(plan["content"]), plan["content"][:40]))
            continue

        doc_id = stable_doc_id(company_code + ":" + domain_path + ":" + source)
        digest = content_hash(text)
        # Remove every historical version of this file in this company. Earlier
        # releases used a different doc_id formula, so doc_id alone is unsafe.
        vectors = emb.embed_texts([plan["content"] for plan in plans])
        if len(vectors) != len(plans) or not plans:
            raise DocumentReadError("分块或向量数量异常，未替换旧版本")
        es.delete_by_query(
            index="kb_finance_doc",
            query={"bool": {"filter": [
                {"term": {"company_code": company_code}},
                {"term": {"source": source}},
                {"term": {"domain_path": domain_path}},
            ]}},
            refresh=True,
            conflicts="proceed",
        )
        now = datetime.now(timezone.utc).isoformat()
        actions = []
        for index, (plan, vector) in enumerate(zip(plans, vectors)):
            chunk_id = "{0}-{1}".format(doc_id, index)
            actions.append({
                "_index": "kb_finance_doc",
                "_id": chunk_id,
                "_source": {
                    "doc_id": doc_id,
                    "document_id": doc_id,
                    "title": path.stem,
                    "source": source,
                    "source_file": source,
                    "chunk_id": chunk_id,
                    "chunk_index": index,
                    "chunk_title": plan.get("title", ""),
                    "section_title": plan.get("section_title", plan.get("title", "")),
                    "section_key": hashlib.sha256((plan.get("section_title", plan.get("title", "")) or "文档正文").encode("utf-8")).hexdigest()[:16],
                    "table_header": plan.get("table_header", ""),
                    "table_index": int(plan.get("table_index") or 0),
                    "table_row_start": int(plan.get("table_row_start") or 0),
                    "table_row_end": int(plan.get("table_row_end") or 0),
                    "document_table_count": integrity["table_count"],
                    "document_table_row_count": integrity["table_row_count"],
                    "is_structured": bool(plan.get("is_structured", False)),
                    "keywords": plan.get("keywords", []),
                    "heading_path": [],
                    "content": plan["content"],
                    "content_vector": vector,
                    "document_type": document_type(path),
                    "company_code": company_code,
                    "domain_path": domain_path,
                    **metadata,
                    "document_version": digest[:12],
                    "acl": "all",
                    "content_hash": digest,
                    "ingested_at": now,
                    "parser": parser_name,
                },
            })
        success, errors = helpers.bulk(es, actions, refresh=True, raise_on_error=False)
        if errors:
            for error in errors[:5]:
                print("  bulk error:", error)
            raise RuntimeError("bulk 写入有 {0} 条失败".format(len(errors)))
        print("  已入库 {0} 块（success={1}）".format(len(actions), success))


if __name__ == "__main__":
    main()
