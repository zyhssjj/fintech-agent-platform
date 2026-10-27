"""Excel-backed V2 platform catalog and strict three-level validation."""

from __future__ import annotations

import json
from pathlib import Path

CATALOG = json.loads((Path(__file__).resolve().parents[1] / "config" / "catalog_v2.json").read_text(encoding="utf-8"))
COMPANIES = {platform["code"]: {**platform, "domains": [leaf["domain_path"] for leaf in platform["leaves"]]} for platform in CATALOG["platforms"]}


def public_companies() -> list[dict]:
    return list(COMPANIES.values())


def get_company(code: str) -> dict | None:
    return COMPANIES.get(code)


def valid_domain(code: str, domain_path: str) -> bool:
    return get_leaf(code, domain_path) is not None


def get_leaf(code: str, domain_path: str) -> dict | None:
    company = get_company(code)
    return next((leaf for leaf in company["leaves"] if leaf["domain_path"] == domain_path), None) if company else None


def domain_metadata(code: str, domain_path: str) -> dict:
    leaf = get_leaf(code, domain_path)
    if not leaf:
        raise ValueError("请选择当前平台的完整三级目录")
    return {"catalog_version": 2, "domain_level1": leaf["level1"], "domain_level2": leaf["level2"], "domain_level3": leaf["level3"], "domain_path": domain_path, "domain_leaf_id": leaf["leaf_id"]}


def expand_domains(code: str, requested: list[str] | None) -> list[str]:
    company = get_company(code)
    if not company:
        return []
    available = company["domains"]
    if not requested:
        return list(available)
    expanded = []
    for prefix in requested:
        expanded.extend(path for path in available if path == prefix or path.startswith(prefix + "/"))
    return list(dict.fromkeys(expanded))


def domain_tree(code: str) -> list[dict]:
    company = get_company(code)
    if not company:
        return []
    roots = {}
    for leaf in company["leaves"]:
        root = roots.setdefault(leaf["level1"], {"name": leaf["level1"], "path": leaf["level1"], "children": {}})
        secondary = root["children"].setdefault(leaf["level2"], {"name": leaf["level2"], "path": leaf["level1"] + "/" + leaf["level2"], "definition": leaf["level2_definition"], "children": []})
        secondary["children"].append({**leaf, "name": leaf["level3"], "path": leaf["domain_path"], "children": []})
    return [{**root, "children": list(root["children"].values())} for root in roots.values()]
