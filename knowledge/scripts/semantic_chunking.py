"""Validated LLM-assisted semantic chunk planning.

The model may only group immutable source blocks.  It never returns rewritten
document content.  Every response is validated for complete, ordered, exact
coverage before the resulting chunks are accepted.
"""

import json
import re


SEMANTIC_PROFILES = {
    "finance": {"soft_min": 300, "target": 600, "hard_max": 1200},
}
MAX_WINDOW_CHARACTERS = 18_000
MAX_WINDOW_BLOCKS = 120


class SemanticChunkingError(RuntimeError):
    """Raised when a semantic plan is unusable and must fall back."""


def split_long_unit(value, size):
    pieces = []
    start = 0
    while start < len(value):
        end = min(start + size, len(value))
        if end < len(value):
            boundary = max(
                value.rfind(mark, start + size // 2, end)
                for mark in "。！？；;\n"
            )
            if boundary >= start + size // 2:
                end = boundary + 1
        piece = value[start:end].strip()
        if piece:
            pieces.append(piece)
        if end >= len(value):
            break
        start = end
    return pieces


def atomize_text(text, hard_max):
    """Create immutable, ordered source blocks from parsed text."""
    units = [line.strip() for line in text.splitlines() if line.strip()]
    atoms = []
    for unit in units:
        pieces = split_long_unit(unit, hard_max) if len(unit) > hard_max else [unit]
        for piece in pieces:
            atoms.append(
                {
                    "id": "B{0:05d}".format(len(atoms) + 1),
                    "content": piece,
                }
            )
    return atoms


def window_atoms(atoms):
    windows = []
    current = []
    characters = 0
    for atom in atoms:
        size = len(atom["content"]) + 1
        if current and (
            len(current) >= MAX_WINDOW_BLOCKS
            or characters + size > MAX_WINDOW_CHARACTERS
        ):
            windows.append(current)
            current = []
            characters = 0
        current.append(atom)
        characters += size
    if current:
        windows.append(current)
    return windows


def extract_json_object(value):
    text = (value or "").strip()
    if text.startswith("```"):
        text = re.sub(r"^```(?:json)?\s*", "", text, flags=re.IGNORECASE)
        text = re.sub(r"\s*```$", "", text)
    start = text.find("{")
    end = text.rfind("}")
    if start < 0 or end <= start:
        raise SemanticChunkingError("语义切块模型没有返回有效JSON")
    try:
        return json.loads(text[start : end + 1])
    except json.JSONDecodeError as error:
        raise SemanticChunkingError("语义切块模型返回的JSON无法解析") from error


def planning_prompt(collection, window, profile, window_index, window_count):
    blocks = "\n".join(
        "[{0}] {1}".format(atom["id"], atom["content"])
        for atom in window
    )
    system = (
        "你是企业知识库语义切块规划器。你只能决定给定原文块的组合边界，"
        "不得改写、摘要、删除、复制或调整任何原文块。必须让每个块恰好出现一次，"
        "保持原顺序。只返回JSON，不要返回Markdown。"
    )
    user = (
        "知识域：{collection}\n窗口：{window_index}/{window_count}\n"
        "建议长度：{soft_min}-{hard_max}字符，目标约{target}字符；"
        "硬上限{hard_max}字符。优先保持标题、条款、列表、表格及其说明完整。\n"
        "返回格式：{{\"chunks\":[{{\"start_id\":\"B00001\","
        "\"end_id\":\"B00003\",\"title\":\"主题短标题\","
        "\"keywords\":[\"关键词\"]}}]}}\n\n原文块：\n{blocks}"
    ).format(
        collection=collection.upper(),
        window_index=window_index,
        window_count=window_count,
        soft_min=profile["soft_min"],
        target=profile["target"],
        hard_max=profile["hard_max"],
        blocks=blocks,
    )
    return system, user


def validate_window_plan(window, response, hard_max):
    payload = extract_json_object(response)
    proposed = payload.get("chunks")
    if not isinstance(proposed, list) or not proposed:
        raise SemanticChunkingError("语义切块方案没有包含任何分块")

    positions = {atom["id"]: index for index, atom in enumerate(window)}
    expected = 0
    accepted = []
    for item in proposed:
        if not isinstance(item, dict):
            raise SemanticChunkingError("语义切块方案格式错误")
        start_id = str(item.get("start_id", ""))
        end_id = str(item.get("end_id", ""))
        if start_id not in positions or end_id not in positions:
            raise SemanticChunkingError("语义切块方案引用了不存在的原文块")
        start = positions[start_id]
        end = positions[end_id]
        if start != expected or end < start:
            raise SemanticChunkingError("语义切块方案存在遗漏、重复或乱序")
        source_atoms = window[start : end + 1]
        content = "\n".join(atom["content"] for atom in source_atoms).strip()
        if len(content) > hard_max:
            raise SemanticChunkingError("语义切块方案超过硬长度上限")
        if len(content) < 30:
            raise SemanticChunkingError("语义切块方案产生了无效短分块")
        accepted.append(
            {
                "content": content,
                "start_id": start_id,
                "end_id": end_id,
                "title": str(item.get("title", "")).strip()[:200],
                "keywords": [
                    str(value).strip()[:50]
                    for value in item.get("keywords", [])[:12]
                    if str(value).strip()
                ]
                if isinstance(item.get("keywords", []), list)
                else [],
            }
        )
        expected = end + 1
    if expected != len(window):
        raise SemanticChunkingError("语义切块方案没有覆盖全部原文块")
    return accepted


def plan_semantic_chunks(text, collection, call_model, progress=None):
    """Return validated chunks using a caller-supplied LLM function.

    ``call_model`` receives ``(system_prompt, user_prompt)`` and returns text.
    ``progress`` optionally receives ``(stage, message)``.
    """
    if collection not in SEMANTIC_PROFILES:
        raise SemanticChunkingError("当前知识域不支持语义增强入库")
    profile = SEMANTIC_PROFILES[collection]
    atoms = atomize_text(text, profile["hard_max"])
    if not atoms:
        raise SemanticChunkingError("文档没有可供语义分析的结构单元")
    windows = window_atoms(atoms)
    plans = []
    for index, window in enumerate(windows, start=1):
        if progress:
            progress(
                "semantic_analysis",
                "正在分析语义窗口 {0}/{1}".format(index, len(windows)),
            )
        system, user = planning_prompt(
            collection,
            window,
            profile,
            index,
            len(windows),
        )
        response = call_model(system, user)
        if progress:
            progress(
                "plan_validation",
                "正在校验语义切块方案 {0}/{1}".format(index, len(windows)),
            )
        plans.extend(validate_window_plan(window, response, profile["hard_max"]))
    reconstructed = "\n".join(plan["content"] for plan in plans)
    atom_source = "\n".join(atom["content"] for atom in atoms)
    if reconstructed != atom_source:
        raise SemanticChunkingError("语义切块结果与解析原文不一致")
    return {
        "chunks": [plan["content"] for plan in plans],
        "plans": plans,
        "atom_count": len(atoms),
        "window_count": len(windows),
    }
