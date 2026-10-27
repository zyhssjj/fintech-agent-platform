"""Regression tests for deterministic complete-list pagination."""

from __future__ import annotations

import unittest
from unittest.mock import patch

from scripts import app


def expanded(chunk_id: str, index: int, content: str) -> dict:
    return {
        "id": chunk_id,
        "section_expanded": True,
        "exact_subject_match": True,
        "source": {
            "doc_id": "doc-table",
            "chunk_id": chunk_id,
            "section_title": "防火墙（共3项）",
            "table_header": "序号 | 名称",
            "source": "firewall.md",
            "document_version": "v1",
            "chunk_index": index,
            "content": "【章节】防火墙（共3项）\n【表头】序号 | 名称\n【表格数据】\n" + content,
        },
    }


class CompleteListTests(unittest.TestCase):
    def test_returns_source_rows_in_pages(self):
        candidates = [
            expanded("table-0", 0, "1 | FW-A\n2 | FW-B"),
            expanded("table-1", 1, "3 | FW-C"),
        ]
        class FakeElasticsearch:
            def search(self, **_kwargs):
                return {"hits": {"hits": [{"_source": item["source"]} for item in candidates]}}

        with patch.object(app, "_retrieve_finance", return_value=(candidates, [{"list_query": True, "section_anchor_found": True}])), patch.object(app, "es", return_value=FakeElasticsearch()):
            page = app._complete_list(app.CompleteListRequest(query="防火墙有哪些", page=2, page_size=2))
        self.assertTrue(page["available"])
        self.assertEqual(page["total"], 3)
        self.assertEqual(page["total_pages"], 2)
        self.assertEqual(page["rows"], ["3 | FW-C"])

    def test_refuses_list_when_no_exact_section_exists(self):
        with patch.object(app, "_retrieve_finance", return_value=([], [{"list_query": True, "section_anchor_found": False}])):
            result = app._complete_list(app.CompleteListRequest(query="测试工具有哪些"))
        self.assertFalse(result["available"])
        self.assertIn("不能保证完整性", result["message"])


if __name__ == "__main__":
    unittest.main()
