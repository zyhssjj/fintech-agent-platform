"""Explicit one-time document reset; never clear users, models or credentials.

Stop the app before running this command against its Elasticsearch service.
"""

from __future__ import annotations

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import shutil


def protected_snapshot(project: Path) -> dict:
    result = {}
    candidates = [project / ".env"]
    runtime = project / "runtime"
    if runtime.exists():
        candidates.extend(path for path in runtime.rglob("*") if "elasticsearch-data" not in path.relative_to(runtime).parts and "ingestion_jobs" not in path.relative_to(runtime).parts)
    for path in candidates:
        if path.is_file():
            result[path.relative_to(project).as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
    return result


def reset_documents(project: Path, client) -> dict:
    from scripts.create_indices import INDEX_DEFINITIONS, index_definition

    project = project.resolve(strict=True)
    root = project / "data" / "finance"
    if root.is_symlink() or root.resolve() != root:
        raise RuntimeError("Unsafe finance directory; refusing to clear")
    before = protected_snapshot(project)
    old_files = [path.relative_to(project).as_posix() for path in root.rglob("*") if path.is_file()] if root.exists() else []
    old_chunks = client.count(index="kb_finance_doc")["count"] if client.indices.exists(index="kb_finance_doc") else 0
    if client.indices.exists(index="kb_finance_doc"):
        client.indices.delete(index="kb_finance_doc")
    client.indices.create(index="kb_finance_doc", **index_definition(INDEX_DEFINITIONS["kb_finance_doc"]))
    if root.exists():
        shutil.rmtree(root)
    root.mkdir(parents=True)
    after = protected_snapshot(project)
    if after != before:
        raise RuntimeError("Protected runtime configuration changed during reset")
    return {"catalog_version": 2, "at": datetime.now(timezone.utc).isoformat(), "removed_chunks": old_chunks, "removed_files": old_files, "protected_hashes": before, "protected_unchanged": True}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--confirm-clear-old-documents", action="store_true")
    args = parser.parse_args()
    if not args.confirm_clear_old_documents:
        parser.error("Explicit --confirm-clear-old-documents is required")
    from elasticsearch import Elasticsearch
    project = Path(__file__).resolve().parents[1]
    report = reset_documents(project, Elasticsearch(os.getenv("ES_URL", "http://localhost:9202")))
    (project / "runtime" / "catalog_v2_reset.json").write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    print(json.dumps({key: value for key, value in report.items() if key != "protected_hashes"}, ensure_ascii=False))


if __name__ == "__main__":
    main()
