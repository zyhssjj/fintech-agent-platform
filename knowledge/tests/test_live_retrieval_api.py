"""Self-contained live V2 regression: uploads temporary fixtures then deletes them.

Run inside the app container with KNOWLEDGE_TEST_BASE_URL=http://127.0.0.1:8000.
No pre-existing business documents are required or modified.
"""

from __future__ import annotations

import io
import json
import os
import time
import unittest
import uuid
from urllib.error import HTTPError
from urllib.parse import urlencode
from urllib.request import Request, urlopen

from scripts.company_catalog import COMPANIES, domain_metadata
from scripts.reset_catalog_v2 import protected_snapshot
from scripts import app

BASE_URL = os.getenv("KNOWLEDGE_TEST_BASE_URL", "http://127.0.0.1:8201")


def request_json(path, payload=None, method=None, headers=None, raw=None):
    data = raw if raw is not None else json.dumps(payload, ensure_ascii=False).encode("utf-8") if payload is not None else None
    request = Request(BASE_URL + path, data=data, headers=headers or ({"Content-Type": "application/json"} if data else {}), method=method or ("POST" if data else "GET"))
    with urlopen(request, timeout=120) as response:
        return json.loads(response.read().decode("utf-8"))


def upload(code, path, filename, content):
    boundary = "V2-" + uuid.uuid4().hex
    body = b""
    for key, value in (("company_code", code), ("domain_path", path), ("mode", "standard")):
        body += (f"--{boundary}\r\nContent-Disposition: form-data; name=\"{key}\"\r\n\r\n{value}\r\n").encode()
    body += (f"--{boundary}\r\nContent-Disposition: form-data; name=\"files\"; filename=\"{filename}\"\r\nContent-Type: application/octet-stream\r\n\r\n").encode()
    body += content + f"\r\n--{boundary}--\r\n".encode()
    return request_json("/api/ingestion/upload", raw=body, headers={"Content-Type": f"multipart/form-data; boundary={boundary}"})


def wait_job(job):
    for _ in range(180):
        result = request_json("/api/ingestion/jobs/" + job["id"])
        if result["status"] == "succeeded":
            return result
        if result["status"] == "failed":
            raise AssertionError(result["message"])
        time.sleep(1)
    raise AssertionError("ingestion timeout")


def document_list(code, prefix=None):
    params = {"company_code": code}
    if prefix:
        params["domain_paths"] = prefix
    return request_json("/api/documents?" + urlencode(params))["documents"]


class LiveRetrievalApiTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fixtures = []
        cls.prefix = "v2-acceptance-" + uuid.uuid4().hex[:8]
        cls.before = protected_snapshot(app.PROJECT_DIR)
        cls.addClassCleanup(cls.cleanup)
        formats = [".md", ".txt", ".docx", ".pdf"]
        for code, platform in COMPANIES.items():
            roots = list(dict.fromkeys(leaf["level1"] for leaf in platform["leaves"]))
            for root in roots:
                leaf = next(leaf for leaf in platform["leaves"] if leaf["level1"] == root)
                marker = cls.prefix + "-" + str(len(cls.fixtures))
                text = "# 验收资料\n" + marker + " 该资料用于验证平台及三级目录隔离，验收专用信息为蓝鲸审核编号九七三，不能跨平台或者跨目录返回。"
                suffix = formats[len(cls.fixtures) % len(formats)]
                if suffix == ".docx":
                    import docx
                    doc = docx.Document()
                    doc.add_paragraph(text)
                    buffer = io.BytesIO()
                    doc.save(buffer)
                    content = buffer.getvalue()
                elif suffix == ".pdf":
                    import fitz
                    doc = fitz.open()
                    page = doc.new_page()
                    page.insert_text((50, 60), marker + " V2 platform and three-level metadata validation document. Verification reference BLUE-WHALE-973.")
                    content = doc.tobytes()
                else:
                    content = text.encode("utf-8")
                fixture = {"code": code, "leaf": leaf, "marker": marker, "filename": marker + suffix, "content": content}
                cls.fixtures.append(fixture)
                job = upload(code, leaf["domain_path"], fixture["filename"], content)
                wait_job(job)
                cls.assert_metadata(job, code, leaf["domain_path"])
                fixture["doc"] = next(doc for doc in document_list(code) if doc["name"] == marker)

    @classmethod
    def assert_metadata(cls, data, code, path):
        for key, value in domain_metadata(code, path).items():
            if data.get(key) != value:
                raise AssertionError(f"{key}: {data.get(key)} != {value}")

    @classmethod
    def cleanup(cls):
        failures = []
        for code in COMPANIES:
            for doc in document_list(code):
                if doc["name"].startswith(cls.prefix):
                    try:
                        request_json(f"/api/documents/{doc['id']}?" + urlencode({"company_code": code}), method="DELETE")
                    except Exception as exc:
                        failures.append(str(exc))
        if failures:
            raise AssertionError("fixture cleanup failed: " + str(failures))

    def search(self, fixture, prefix, mode="retrieve", code=None):
        return request_json("/api/search", {"query": fixture["marker"] + " 验收资料内容是什么", "company_code": code or fixture["code"], "domain_paths": [prefix], "mode": mode, "top_k": 8})

    def test_health(self):
        self.assertEqual(request_json("/health")["es"], "up")

    def test_three_platform_trees(self):
        for code, platform in COMPANIES.items():
            tree = request_json(f"/api/companies/{code}/tree")
            leaves = [leaf for root in tree["tree"] for secondary in root["children"] for leaf in secondary["children"]]
            self.assertEqual(len(leaves), len(platform["leaves"]))

    def test_four_document_formats_and_metadata(self):
        self.assertEqual({os.path.splitext(item["filename"])[1] for item in self.fixtures}, {".md", ".txt", ".docx", ".pdf"})
        for fixture in self.fixtures:
            self.assert_metadata(fixture["doc"], fixture["code"], fixture["leaf"]["domain_path"])

    def test_document_filters_at_all_three_levels(self):
        for fixture in self.fixtures:
            for depth in (1, 2, 3):
                prefix = "/".join(fixture["leaf"]["domain_path"].split("/")[:depth])
                docs = document_list(fixture["code"], prefix)
                self.assertIn(fixture["doc"]["id"], [doc["id"] for doc in docs])
                self.assertTrue(all(doc["domain_path"] == prefix or doc["domain_path"].startswith(prefix + "/") for doc in docs))
                self.assertTrue(all(doc["company_code"] == fixture["code"] for doc in docs))

    def test_retrieval_at_all_three_levels_all_platforms(self):
        for fixture in self.fixtures:
            for depth in (1, 2, 3):
                prefix = "/".join(fixture["leaf"]["domain_path"].split("/")[:depth])
                result = self.search(fixture, prefix)
                sources = result["sources"]
                self.assertIn(fixture["marker"], "\n".join(source["content"] for source in sources))
                self.assertTrue(all(source["company_code"] == fixture["code"] for source in sources))
                self.assertTrue(all(source["domain_path"] == prefix or source["domain_path"].startswith(prefix + "/") for source in sources))

    def test_cross_platform_query_never_leaks(self):
        fixture = self.fixtures[0]
        for code in COMPANIES:
            if code != fixture["code"]:
                result = self.search(fixture, fixture["leaf"]["level1"], code=code)
                self.assertNotIn(fixture["marker"], "\n".join(source["content"] for source in result["sources"]))
                self.assertTrue(all(source["company_code"] == code for source in result["sources"]))

    def test_cross_level_query_never_leaks(self):
        fixture = self.fixtures[0]
        root = next(leaf["level1"] for leaf in COMPANIES[fixture["code"]]["leaves"] if leaf["level1"] != fixture["leaf"]["level1"])
        result = self.search(fixture, root)
        self.assertNotIn(fixture["marker"], "\n".join(source["content"] for source in result["sources"]))

    def test_answer_keeps_exact_leaf_sources(self):
        fixture = self.fixtures[0]
        result = self.search(fixture, fixture["leaf"]["domain_path"], mode="answer")
        self.assertTrue(result["answer"])
        self.assertEqual(result["answer_mode"], "answer")
        self.assertTrue(result["sources"])
        self.assertTrue(all(source["domain_path"] == fixture["leaf"]["domain_path"] and source["company_code"] == fixture["code"] for source in result["sources"]))

    def test_invalid_upload_and_invalid_query_422(self):
        diff = next(path for path in COMPANIES["neimeng_juyuan"]["domains"] if path not in COMPANIES["beijing_motian"]["domains"])
        for path in ("基金层面", "基金层面/基金设立", "invalid/invalid/invalid", diff):
            with self.assertRaises(HTTPError) as exc:
                upload("beijing_motian", path, "invalid.md", b"invalid")
            self.assertEqual(exc.exception.code, 422)
        with self.assertRaises(HTTPError) as exc:
            request_json("/api/search", {"query": "test", "company_code": "beijing_hangke", "domain_paths": ["invalid"]})
        self.assertEqual(exc.exception.code, 422)

    def test_internal_api_auth_and_scope(self):
        with self.assertRaises(HTTPError) as exc:
            request_json("/api/v1/retrieve", {"query": "test"})
        self.assertEqual(exc.exception.code, 401)
        with self.assertRaises(HTTPError) as exc:
            request_json("/api/v1/retrieve", {"query": "test"}, headers={"Content-Type": "application/json", "Authorization": "Bearer wrong-key"})
        self.assertEqual(exc.exception.code, 401)
        fixture = self.fixtures[0]
        result = request_json("/api/v1/retrieve", {"query": fixture["marker"], "company_code": fixture["code"], "domain_paths": [fixture["leaf"]["domain_path"]]}, headers={"Content-Type": "application/json", "Authorization": "Bearer " + os.environ["KNOWLEDGE_INTERNAL_API_KEY"]})
        self.assertTrue(result["sources"])
        self.assertTrue(all(source["domain_path"] == fixture["leaf"]["domain_path"] for source in result["sources"]))
        answer = request_json("/api/v1/answer", {"query": fixture["marker"] + " 验收资料内容是什么", "company_code": fixture["code"], "domain_paths": [fixture["leaf"]["domain_path"]]}, headers={"Content-Type": "application/json", "Authorization": "Bearer " + os.environ["KNOWLEDGE_INTERNAL_API_KEY"]})
        self.assertEqual(answer["answer_mode"], "answer")
        self.assertTrue(answer["answer"])
        self.assertTrue(all(source["domain_path"] == fixture["leaf"]["domain_path"] for source in answer["sources"]))

    def test_update_same_file_preserves_other_documents(self):
        fixture = self.fixtures[1]
        old_doc = fixture["doc"]
        text = fixture["marker"] + " UPDATED-CONTENT 本次更新验收确认旧版本已替换，其他目录资料保持不变。"
        wait_job(upload(fixture["code"], fixture["leaf"]["domain_path"], fixture["filename"], text.encode("utf-8")))
        docs = document_list(fixture["code"])
        self.assertEqual(sum(doc["name"] == fixture["marker"] for doc in docs), 1)
        doc = next(doc for doc in docs if doc["name"] == fixture["marker"])
        detail = request_json(f"/api/documents/{doc['id']}/detail?" + urlencode({"company_code": fixture["code"], "full_text": "true"}))
        self.assertIn("UPDATED-CONTENT", detail["full_text"])
        for other in self.fixtures:
            self.assertIn(other["doc"]["id"], [item["id"] for item in document_list(other["code"])])

    def test_wrong_platform_detail_delete_return_404(self):
        fixture = self.fixtures[0]
        code = "beijing_motian"
        for suffix, method in (("/detail", "GET"), ("", "DELETE")):
            with self.assertRaises(HTTPError) as exc:
                request_json(f"/api/documents/{fixture['doc']['id']}{suffix}?" + urlencode({"company_code": code}), method=method)
            self.assertEqual(exc.exception.code, 404)

    def test_same_filename_across_leaves_and_platforms_deletion_isolated(self):
        name = self.prefix + "-same"
        scopes = [("beijing_hangke", COMPANIES["beijing_hangke"]["leaves"][0]), ("beijing_hangke", COMPANIES["beijing_hangke"]["leaves"][1]), ("beijing_motian", COMPANIES["beijing_motian"]["leaves"][0])]
        docs = []
        for index, (code, leaf) in enumerate(scopes):
            wait_job(upload(code, leaf["domain_path"], name + ".md", (name + f" UNIQUE-SCOPE-{index} 同名文件隔离验收资料，不得覆盖另一个平台或目录。").encode()))
            doc = next(doc for doc in document_list(code, leaf["domain_path"]) if doc["name"] == name)
            docs.append((code, leaf, doc))
        self.assertEqual(len({doc["id"] for _, _, doc in docs}), 3)
        first_code, _, first = docs[0]
        deletion = request_json(f"/api/documents/{first['id']}?" + urlencode({"company_code": first_code}), method="DELETE")
        self.assertTrue(deletion["raw_file_deleted"])
        self.assertNotIn(first["id"], [doc["id"] for doc in document_list(first_code)])
        for code, leaf, doc in docs[1:]:
            self.assertIn(doc["id"], [item["id"] for item in document_list(code, leaf["domain_path"])])

    def test_protected_configuration_unchanged(self):
        # Audit is append-only as expected; all actual runtime settings stay byte-identical.
        after = protected_snapshot(app.PROJECT_DIR)
        before = dict(self.before)
        for key in ("runtime/document_audit.jsonl",):
            before.pop(key, None)
            after.pop(key, None)
        self.assertEqual(before, after)


if __name__ == "__main__":
    unittest.main()
