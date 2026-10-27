"""V2 catalog, upload validation, metadata and reset boundary regression tests."""

from __future__ import annotations

import io
import json
import tempfile
from pathlib import Path
import unittest
from unittest.mock import patch, Mock

from fastapi import BackgroundTasks, UploadFile, HTTPException
from fastapi.testclient import TestClient
from scripts import app, retrieval
from scripts.auth_store import AuthStore
from scripts.company_catalog import CATALOG, COMPANIES, domain_metadata, domain_tree, expand_domains, get_leaf, valid_domain
from scripts.reset_catalog_v2 import protected_snapshot, reset_documents


class CatalogTests(unittest.TestCase):
    def test_exact_platforms_names_codes_counts(self):
        self.assertEqual([(code, item["name"], len(item["leaves"])) for code, item in COMPANIES.items()], [("beijing_hangke", "航科君富", 38), ("beijing_motian", "膜天君富", 34), ("neimeng_juyuan", "聚元私募", 39)])

    def test_all_111_leaves_complete_unique_metadata(self):
        ids = set()
        for code, platform in COMPANIES.items():
            for leaf in platform["leaves"]:
                with self.subTest(code=code, path=leaf["domain_path"]):
                    self.assertEqual(leaf["domain_path"], "/".join(leaf[key] for key in ("level1", "level2", "level3")))
                    self.assertTrue(leaf["level2_definition"])
                    self.assertTrue(leaf["related_files"])
                    self.assertEqual(leaf["related_files"], [item["name"] for item in leaf["file_items"]])
                    self.assertEqual(get_leaf(code, leaf["domain_path"]), leaf)
                    self.assertNotIn(leaf["leaf_id"], ids)
                    ids.add(leaf["leaf_id"])
                    self.assertEqual(domain_metadata(code, leaf["domain_path"])["domain_leaf_id"], leaf["leaf_id"])
        self.assertEqual(len(ids), 111)

    def test_platform_differences_are_not_shared(self):
        sets = [set(item["domains"]) for item in COMPANIES.values()]
        self.assertNotEqual(sets[0], sets[1])
        self.assertNotEqual(sets[0], sets[2])
        self.assertNotEqual(sets[1], sets[2])

    def test_prefix_expansion_at_every_level(self):
        for code, platform in COMPANIES.items():
            for leaf in platform["leaves"]:
                for depth in (1, 2, 3):
                    prefix = "/".join(leaf["domain_path"].split("/")[:depth])
                    expected = [path for path in platform["domains"] if path == prefix or path.startswith(prefix + "/")]
                    self.assertEqual(expand_domains(code, [prefix]), expected)

    def test_non_leaf_old_path_and_unknown_company_rejected(self):
        for path in ("基金层面", "基金层面/基金设立", "基金层面/基金运营", "公司层面/公司运营类", "不存在/目录/事项"):
            self.assertFalse(valid_domain("beijing_hangke", path))
        self.assertFalse(valid_domain("future_intelligence", "公司层面/公司业务文档类"))

    def test_tree_matches_all_leaves_and_definitions(self):
        for code, platform in COMPANIES.items():
            tree = domain_tree(code)
            leaves = [leaf for root in tree for secondary in root["children"] for leaf in secondary["children"]]
            self.assertEqual([leaf["path"] for leaf in leaves], platform["domains"])
            self.assertTrue(all(secondary["definition"] for root in tree for secondary in root["children"]))

    def test_tree_api_and_cross_platform_paths(self):
        with TestClient(app.app) as client:
            for code, platform in COMPANIES.items():
                response = client.get(f"/api/companies/{code}/tree")
                self.assertEqual(response.status_code, 200)
                self.assertEqual(response.json()["tree"], domain_tree(code))
            self.assertEqual(client.get("/api/companies/future_intelligence/tree").status_code, 422)
            missing_platform = client.post("/api/ingestion/upload", data={"domain_path": COMPANIES["beijing_hangke"]["domains"][0]}, files={"files": ("test.md", b"test")})
            self.assertEqual(missing_platform.status_code, 422)
            diff = next(path for path in COMPANIES["neimeng_juyuan"]["domains"] if path not in COMPANIES["beijing_motian"]["domains"])
            for path in (diff, "基金层面", "基金层面/基金设立", "x/y/z"):
                response = client.post("/api/ingestion/upload", data={"company_code": "beijing_motian", "domain_path": path}, files={"files": ("test.md", b"test")})
                self.assertEqual(response.status_code, 422, path)

    def test_retrieve_filters_all_selected_leaves_in_one_call(self):
        with patch.object(app, "es"), patch.object(app, "emb"), patch.object(retrieval, "retrieve", return_value=([], {})) as retrieve:
            app._retrieve_finance(app.RetrieveRequest(query="test", company_code="beijing_motian", domain_paths=["基金层面"]), 8)
            self.assertEqual(retrieve.call_count, 1)
            filters = retrieve.call_args.kwargs["filters"]
            self.assertEqual(filters["company_code"], "beijing_motian")
            self.assertEqual(filters["domain_path"], expand_domains("beijing_motian", ["基金层面"]))
            self.assertIn({"terms": {"domain_path": filters["domain_path"]}}, retrieval._filters(filters))

    def test_invalid_query_scopes_fail_before_es(self):
        for code, scopes in [("future_intelligence", None), ("beijing_hangke", ["公司层面/公司运营类"]), ("beijing_hangke", ["基金层面", "invalid"])]:
            with self.assertRaises(HTTPException) as exc:
                app._finance_filters(code, scopes)
            self.assertEqual(exc.exception.status_code, 422)

    def test_credential_scope_normalizes_parent_to_leaf_permissions(self):
        company = COMPANIES["beijing_hangke"]
        root = company["leaves"][0]["level1"]
        scopes = app._normalize_knowledge_scopes([{"company_code": "beijing_hangke", "domain_paths": [root]}])
        self.assertEqual(scopes, [{"company_code": "beijing_hangke", "domain_paths": expand_domains("beijing_hangke", [root])}])
        with self.assertRaises(HTTPException) as exc:
            app._normalize_knowledge_scopes([{"company_code": "beijing_hangke", "domain_paths": ["无效/目录"]}])
        self.assertEqual(exc.exception.status_code, 422)

    def test_credential_scope_enforces_leaf_and_platform_boundaries(self):
        allowed_leaf = COMPANIES["beijing_hangke"]["leaves"][0]["domain_path"]
        denied_leaf = COMPANIES["beijing_hangke"]["leaves"][1]["domain_path"]
        client = {"knowledge_scopes": [{"company_code": "beijing_hangke", "domain_paths": [allowed_leaf]}]}

        implicit = app.RetrieveRequest(query="test", company_code="beijing_hangke")
        app._authorize_request_scope(client, implicit)
        self.assertEqual(implicit.domain_paths, [allowed_leaf])

        for code, paths in (("beijing_hangke", [denied_leaf]), ("beijing_motian", None)):
            with self.subTest(code=code, paths=paths), self.assertRaises(HTTPException) as exc:
                app._authorize_request_scope(client, app.RetrieveRequest(query="test", company_code=code, domain_paths=paths))
            self.assertEqual(exc.exception.status_code, 403)

    def test_create_credential_persists_leaf_level_scopes(self):
        leaf = COMPANIES["beijing_motian"]["leaves"][0]
        with tempfile.TemporaryDirectory() as tmp, patch.object(app, "CREDENTIALS_PATH", Path(tmp) / "credentials.json"):
            with TestClient(app.app) as client:
                response = client.post("/api/credentials", json={
                    "name": "三级目录测试智能体",
                    "knowledge_scopes": [{"company_code": "beijing_motian", "domain_paths": [leaf["domain_path"]]}],
                    "can_answer": True,
                    "rate_limit_per_min": 80,
                })
            self.assertEqual(response.status_code, 200)
            created = response.json()
            self.assertEqual(created["knowledge_scopes"], [{"company_code": "beijing_motian", "domain_paths": [leaf["domain_path"]]}])
            self.assertTrue(created["api_key"].startswith("wk_"))
            stored = json.loads((Path(tmp) / "credentials.json").read_text(encoding="utf-8"))["clients"][0]
            self.assertNotIn("api_key", stored)
            self.assertEqual(stored["api_key_sha256"], app._sha256(created["api_key"]))

    def test_machine_api_applies_credential_scope_end_to_end(self):
        allowed = COMPANIES["beijing_hangke"]["leaves"][0]["domain_path"]
        denied = COMPANIES["beijing_hangke"]["leaves"][1]["domain_path"]
        with tempfile.TemporaryDirectory() as tmp, patch.object(app, "CREDENTIALS_PATH", Path(tmp) / "credentials.json"), patch.object(app, "es"), patch.object(app, "emb"), patch.object(retrieval, "retrieve", return_value=([], {})) as retrieve:
            with TestClient(app.app) as client:
                created = client.post("/api/credentials", json={"name": "scope-test", "knowledge_scopes": [{"company_code": "beijing_hangke", "domain_paths": [allowed]}]}).json()
                headers = {"Authorization": "Bearer " + created["api_key"]}
                response = client.post("/api/v1/retrieve", json={"query": "test", "company_code": "beijing_hangke"}, headers=headers)
                self.assertEqual(response.status_code, 200)
                self.assertEqual(response.json()["routed_to"], [allowed])
                self.assertEqual(retrieve.call_args.kwargs["filters"]["domain_path"], [allowed])
                self.assertEqual(client.post("/api/v1/retrieve", json={"query": "test", "company_code": "beijing_hangke", "domain_paths": [denied]}, headers=headers).status_code, 403)
                self.assertEqual(client.post("/api/v1/retrieve", json={"query": "test", "company_code": "beijing_motian"}, headers=headers).status_code, 403)

    def test_user_api_key_merges_multiple_bound_credentials_and_updates_immediately(self):
        hangke_leaf = COMPANIES["beijing_hangke"]["leaves"][0]["domain_path"]
        motian_leaf = COMPANIES["beijing_motian"]["leaves"][0]["domain_path"]
        with tempfile.TemporaryDirectory() as tmp, \
                patch.object(app, "CREDENTIALS_PATH", Path(tmp) / "credentials.json"), \
                patch.object(app, "auth_store", AuthStore(Path(tmp))), \
                patch.object(app, "es"), patch.object(app, "emb"), \
                patch.object(retrieval, "retrieve", return_value=([], {})):
            with TestClient(app.app) as client:
                first = client.post("/api/credentials", json={
                    "name": "航科检索",
                    "knowledge_scopes": [{"company_code": "beijing_hangke", "domain_paths": [hangke_leaf]}],
                    "rate_limit_per_min": 50,
                }).json()
                second = client.post("/api/credentials", json={
                    "name": "膜天问答",
                    "knowledge_scopes": [{"company_code": "beijing_motian", "domain_paths": [motian_leaf]}],
                    "can_answer": True,
                    "rate_limit_per_min": 20,
                }).json()
                created = client.post("/api/users", json={
                    "name": "外部调用用户",
                    "position": "测试岗",
                    "username": "external-user",
                    "password": "Password123",
                    "role": "user",
                    "credential_ids": [first["id"], second["id"]],
                })
                self.assertEqual(created.status_code, 201)
                user = created.json()
                self.assertTrue(user["api_key"].startswith("wk_user_"))
                self.assertEqual(user["credential_ids"], [first["id"], second["id"]])
                headers = {"Authorization": "Bearer " + user["api_key"]}

                for code, leaf in (("beijing_hangke", hangke_leaf), ("beijing_motian", motian_leaf)):
                    response = client.post("/api/v1/retrieve", json={"query": "test", "company_code": code}, headers=headers)
                    self.assertEqual(response.status_code, 200)
                    self.assertEqual(response.json()["routed_to"], [leaf])
                effective = app.require_client(headers["Authorization"])
                self.assertTrue(effective["can_answer"])
                self.assertEqual(effective["rate_limit_per_min"], 20)
                self.assertEqual(
                    client.post("/api/v1/answer", json={"query": "test", "company_code": "beijing_hangke"}, headers=headers).status_code,
                    403,
                )

                updated = client.put(f"/api/users/{user['id']}", json={"credential_ids": [first["id"]]})
                self.assertEqual(updated.status_code, 200)
                self.assertEqual(updated.json()["credential_ids"], [first["id"]])
                denied = client.post("/api/v1/retrieve", json={"query": "test", "company_code": "beijing_motian"}, headers=headers)
                self.assertEqual(denied.status_code, 403)

                self.assertEqual(client.delete(f"/api/credentials/{first['id']}").status_code, 200)
                self.assertEqual(client.get("/api/users").json()["users"][0]["credential_ids"], [])
                no_scope = client.post("/api/v1/retrieve", json={"query": "test", "company_code": "beijing_hangke"}, headers=headers)
                self.assertEqual(no_scope.status_code, 403)

    def test_same_filename_different_leaves_isolated_and_job_metadata(self):
        import asyncio
        with tempfile.TemporaryDirectory() as tmp, patch.object(app, "DATA_DIR", Path(tmp)), patch.object(app, "_write_job"):
            paths = COMPANIES["beijing_hangke"]["domains"][:2]
            for index, path in enumerate(paths):
                tasks = BackgroundTasks()
                job = asyncio.run(app.console_ingestion_upload(tasks, "beijing_hangke", path, "standard", [UploadFile(io.BytesIO(str(index).encode()), filename="same.md")]))
                leaf = get_leaf("beijing_hangke", path)
                raw = Path(tmp) / "finance" / "beijing_hangke" / leaf["leaf_id"] / "same.md"
                self.assertEqual(raw.read_bytes(), str(index).encode())
                self.assertEqual(job["domain_leaf_id"], leaf["leaf_id"])
                self.assertEqual(job["catalog_version"], 2)
                self.assertEqual(tasks.tasks[0].args[2], path)

    def test_document_center_and_detail_include_over_1000_chunks(self):
        leaf = COMPANIES["beijing_hangke"]["leaves"][0]
        chunks = [{"_id": str(index), "_source": {"doc_id": "doc", "chunk_index": index, "source": "finance/test.md", "content": "content", "company_code": "beijing_hangke", "domain_path": leaf["domain_path"], **domain_metadata("beijing_hangke", leaf["domain_path"])}} for index in range(1439)]
        with patch.object(app, "es"), patch.object(app, "scan", side_effect=lambda *args, **kwargs: iter(chunks)):
            docs = app.console_documents("beijing_hangke", [leaf["level1"]])["documents"]
            self.assertEqual(docs[0]["units"], 1439)
            detail = app.console_document_detail("doc", "beijing_hangke", "", 180, 8, True)
            self.assertEqual(detail["integrity"]["chunk_total"], 1439)
            self.assertTrue(detail["integrity"]["ok"])
            self.assertEqual(detail["document"]["domain_leaf_id"], leaf["leaf_id"])


class ResetBoundaryTests(unittest.TestCase):
    def test_symlinked_finance_directory_refused_before_es_changes(self):
        with tempfile.TemporaryDirectory() as tmp:
            project = Path(tmp).resolve()
            (project / "data").mkdir()
            (project / "outside").mkdir()
            (project / "data/finance").symlink_to(project / "outside", target_is_directory=True)
            client = Mock()
            with self.assertRaises(RuntimeError):
                reset_documents(project, client)
            client.indices.delete.assert_not_called()

    def test_only_finance_files_and_named_index_removed(self):
        with tempfile.TemporaryDirectory() as tmp:
            project = Path(tmp).resolve()
            for name in (".env", "runtime/auth/users.json", "runtime/credentials.json", "runtime/model_config.json", "runtime/master.key", "runtime/document_audit.jsonl", "data/reference/keep.txt", "data/finance/company/old.docx"):
                path = project / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("unchanged", encoding="utf-8")
            before = protected_snapshot(project)
            client = Mock()
            client.indices.exists.return_value = True
            client.count.return_value = {"count": 164}
            report = reset_documents(project, client)
            self.assertEqual(before, protected_snapshot(project))
            self.assertEqual(list((project / "data/finance").iterdir()), [])
            self.assertTrue((project / "data/reference/keep.txt").exists())
            client.indices.delete.assert_called_once_with(index="kb_finance_doc")
            self.assertEqual(client.indices.create.call_args.kwargs["index"], "kb_finance_doc")
            self.assertEqual(report["removed_chunks"], 164)
            self.assertTrue(report["protected_unchanged"])


if __name__ == "__main__":
    unittest.main()
