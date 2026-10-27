"""Encrypted registry for local and cloud reranking endpoints."""

import base64
import hashlib
import json
import os
import threading
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path

from cryptography.fernet import Fernet, InvalidToken


VALID_CATEGORIES = {"cloud", "local"}
VALID_DOMAINS = {"基金层面", "项目层面", "公司层面"}
VALID_PROVIDERS = {"aliyun-qwen3", "compatible-rerank", "local-cross-encoder"}
DASHSCOPE_NATIVE_PATH = "/api/v1/services/rerank/text-rerank/text-rerank"


def utc_now():
    return datetime.now(timezone.utc).isoformat()


def positive_env_int(name, default):
    try:
        return max(1, int(os.getenv(name, default)))
    except (TypeError, ValueError):
        return default


class RerankRegistry:
    def __init__(self, runtime_dir):
        self.directory = Path(runtime_dir) / "rerank_config"
        self.directory.mkdir(parents=True, exist_ok=True)
        self.config_path = self.directory / "models.json"
        self.audit_path = self.directory / "audit.jsonl"
        self.key_path = self.directory / "master.key"
        self.lock = threading.RLock()
        self.cipher = Fernet(self._load_master_key())
        self.audit_retention_days = positive_env_int("RERANK_AUDIT_RETENTION_DAYS", 180)
        self.audit_max_records = positive_env_int("RERANK_AUDIT_MAX_RECORDS", 100_000)
        self.audit_max_bytes = positive_env_int("RERANK_AUDIT_MAX_BYTES", 50 * 1024 * 1024)
        self._last_audit_cleanup = datetime.min.replace(tzinfo=timezone.utc)
        if not self.config_path.exists():
            self._write(self._default_registry())

    def _default_registry(self):
        now = utc_now()
        model_id = "builtin-local-reranker"
        return {
            "version": 1,
            "active_model_id": model_id,
            "models": [
                {
                    "id": model_id,
                    "name": "本地 BGE 重排模型",
                    "category": "local",
                    "provider": "local-cross-encoder",
                    "base_url": "local://bge-reranker-v2-m3",
                    "api_key_encrypted": None,
                    "model": "bge-reranker-v2-m3",
                    "domains": ["基金层面", "项目层面", "公司层面"],
                    "timeout": 120,
                    "candidate_limit": 64,
                    "created_at": now,
                    "updated_at": now,
                    "last_test_status": "success",
                    "last_tested_at": now,
                    "last_test_error": None,
                }
            ],
        }

    def _load_master_key(self):
        configured = (
            os.getenv("RERANK_CONFIG_MASTER_KEY", "").strip()
            or os.getenv("LLM_CONFIG_MASTER_KEY", "").strip()
        )
        if configured:
            encoded = configured.encode("utf-8")
            try:
                Fernet(encoded)
                return encoded
            except (ValueError, TypeError):
                return base64.urlsafe_b64encode(hashlib.sha256(encoded).digest())
        if self.key_path.exists():
            return self.key_path.read_bytes().strip()
        key = Fernet.generate_key()
        self.key_path.write_bytes(key)
        try:
            self.key_path.chmod(0o600)
        except OSError:
            pass
        return key

    def _load(self):
        try:
            return json.loads(self.config_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as error:
            raise RuntimeError("重排模型配置无法读取") from error

    def _write(self, data):
        temporary = self.directory / ".models.tmp"
        temporary.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")
        os.replace(temporary, self.config_path)

    def _encrypt(self, value):
        return self.cipher.encrypt(value.encode("utf-8")).decode("ascii") if value else None

    def _decrypt(self, value):
        if not value:
            return ""
        try:
            return self.cipher.decrypt(value.encode("ascii")).decode("utf-8")
        except InvalidToken as error:
            raise RuntimeError("重排API密钥无法解密，请检查主密钥") from error

    def _normalize(self, payload, existing=None):
        existing = existing or {}
        category = str(payload.get("category", existing.get("category", ""))).strip().lower()
        provider = str(payload.get("provider", existing.get("provider", ""))).strip()
        if category not in VALID_CATEGORIES:
            raise ValueError("重排类型必须是云端模型或本地模型")
        if provider not in VALID_PROVIDERS:
            raise ValueError("不支持的重排服务类型")
        if category == "cloud" and provider == "local-cross-encoder":
            raise ValueError("云端重排不能使用本地CrossEncoder类型")
        if category == "local" and provider != "local-cross-encoder":
            raise ValueError("本地重排当前只支持本地CrossEncoder")
        domains = sorted(set(payload.get("domains", existing.get("domains", []))))
        if not domains or not set(domains).issubset(VALID_DOMAINS):
            raise ValueError("请至少选择一个有效知识域")
        if category == "cloud" and "c3" in domains:
            raise ValueError("云端重排模型禁止获得C3权限")
        name = str(payload.get("name", existing.get("name", ""))).strip()
        model = str(payload.get("model", existing.get("model", ""))).strip()
        base_url = str(payload.get("base_url", existing.get("base_url", ""))).strip().rstrip("/")
        if not name or not model:
            raise ValueError("配置名称和实际模型名称不能为空")
        if category == "cloud" and not base_url.startswith("https://"):
            raise ValueError("云端重排API必须使用HTTPS地址")
        if provider == "aliyun-qwen3":
            if not base_url.endswith(DASHSCOPE_NATIVE_PATH):
                raise ValueError(
                    "阿里云Qwen-Rerank请填写官网完整DashScope接口地址"
                )
            if model != "qwen3-vl-rerank":
                raise ValueError("阿里云Qwen-Rerank实际模型名称必须是qwen3-vl-rerank")
        if category == "local" and not base_url:
            base_url = "local://bge-reranker-v2-m3"
        api_key = payload.get("api_key")
        encrypted = existing.get("api_key_encrypted")
        if payload.get("clear_api_key"):
            encrypted = None
        elif api_key:
            encrypted = self._encrypt(api_key)
        if category == "cloud" and not encrypted:
            raise ValueError("云端重排模型必须配置API Key")
        timeout = int(payload.get("timeout", existing.get("timeout", 12)))
        candidate_limit = int(payload.get("candidate_limit", existing.get("candidate_limit", 48)))
        if not 3 <= timeout <= 120:
            raise ValueError("重排超时必须在3到120秒之间")
        if not 4 <= candidate_limit <= 128:
            raise ValueError("候选上限必须在4到128之间")
        if provider == "aliyun-qwen3" and candidate_limit > 100:
            raise ValueError("阿里云qwen3-vl-rerank文本候选上限不能超过100")
        return {
            "name": name,
            "category": category,
            "provider": provider,
            "base_url": base_url,
            "api_key_encrypted": encrypted,
            "model": model,
            "domains": domains,
            "timeout": timeout,
            "candidate_limit": candidate_limit,
        }

    def _public(self, model, active_id):
        item = {key: value for key, value in model.items() if key != "api_key_encrypted"}
        secret = self._decrypt(model.get("api_key_encrypted"))
        item["has_api_key"] = bool(secret)
        item["api_key_masked"] = "••••••••{0}".format(secret[-4:]) if secret else "无需密钥"
        item["is_active"] = model["id"] == active_id
        return item

    def list_public(self):
        with self.lock:
            data = self._load()
            return [self._public(model, data.get("active_model_id")) for model in data["models"]]

    def create(self, payload):
        with self.lock:
            data = self._load()
            normalized = self._normalize(payload)
            now = utc_now()
            model = {
                "id": uuid.uuid4().hex,
                **normalized,
                "created_at": now,
                "updated_at": now,
                "last_test_status": "untested",
                "last_tested_at": None,
                "last_test_error": None,
            }
            data["models"].append(model)
            self._write(data)
            return self._public(model, data.get("active_model_id"))

    def update(self, model_id, payload):
        with self.lock:
            data = self._load()
            for index, model in enumerate(data["models"]):
                if model["id"] != model_id:
                    continue
                normalized = self._normalize(payload, model)
                changed = any(
                    normalized[key] != model.get(key)
                    for key in ("provider", "base_url", "api_key_encrypted", "model")
                )
                updated = {**model, **normalized, "updated_at": utc_now()}
                if changed:
                    updated.update(last_test_status="untested", last_tested_at=None, last_test_error=None)
                    if data.get("active_model_id") == model_id:
                        data["active_model_id"] = None
                data["models"][index] = updated
                self._write(data)
                return self._public(updated, data.get("active_model_id"))
        raise KeyError(model_id)

    def delete(self, model_id):
        if model_id == "builtin-local-reranker":
            raise ValueError("内置本地重排模型不能删除")
        with self.lock:
            data = self._load()
            before = len(data["models"])
            data["models"] = [item for item in data["models"] if item["id"] != model_id]
            if len(data["models"]) == before:
                raise KeyError(model_id)
            if data.get("active_model_id") == model_id:
                data["active_model_id"] = None
            self._write(data)

    def get_private(self, model_id):
        with self.lock:
            data = self._load()
            model = next((item for item in data["models"] if item["id"] == model_id), None)
            if not model:
                raise KeyError(model_id)
            return {**model, "api_key": self._decrypt(model.get("api_key_encrypted"))}

    def mark_test(self, model_id, success, error=None):
        with self.lock:
            data = self._load()
            for index, model in enumerate(data["models"]):
                if model["id"] != model_id:
                    continue
                updated = {
                    **model,
                    "last_test_status": "success" if success else "failed",
                    "last_tested_at": utc_now(),
                    "last_test_error": error,
                }
                data["models"][index] = updated
                if not success and data.get("active_model_id") == model_id:
                    data["active_model_id"] = None
                self._write(data)
                return self._public(updated, data.get("active_model_id"))
        raise KeyError(model_id)

    def activate(self, model_id):
        with self.lock:
            data = self._load()
            model = next((item for item in data["models"] if item["id"] == model_id), None)
            if not model:
                raise KeyError(model_id)
            if model.get("last_test_status") != "success":
                raise ValueError("重排模型必须先通过连接测试")
            data["active_model_id"] = model_id
            self._write(data)
            return self._public(model, model_id)

    def active_private(self):
        with self.lock:
            data = self._load()
            active_id = data.get("active_model_id")
            model = next((item for item in data["models"] if item["id"] == active_id), None)
            if not model:
                return None
            return {**model, "api_key": self._decrypt(model.get("api_key_encrypted"))}

    def public_status(self):
        active = self.active_private()
        if not active:
            return {"configured": bool(self.list_public()), "active": None}
        public = {key: value for key, value in active.items() if key not in {"api_key", "api_key_encrypted"}}
        public["is_active"] = True
        public["has_api_key"] = bool(active.get("api_key"))
        return {"configured": True, "active": public}

    def audit_call(self, model, domains, candidates, success, latency_ms, error=None, fallback=None, usage_tokens=None):
        record = {
            "timestamp": utc_now(),
            "model_id": model.get("id"),
            "model_name": model.get("name"),
            "category": model.get("category"),
            "provider": model.get("provider"),
            "domains": sorted(set(domains)),
            "candidates": candidates,
            "usage_tokens": usage_tokens,
            "latency_ms": latency_ms,
            "success": bool(success),
            "fallback": fallback,
            "error": error,
        }
        with self.lock:
            with self.audit_path.open("a", encoding="utf-8") as handle:
                handle.write(json.dumps(record, ensure_ascii=False) + "\n")
            now = datetime.now(timezone.utc)
            if now - self._last_audit_cleanup >= timedelta(hours=1):
                self.cleanup_audit()

    def cleanup_audit(self):
        cutoff = datetime.now(timezone.utc) - timedelta(days=self.audit_retention_days)
        with self.lock:
            self._last_audit_cleanup = datetime.now(timezone.utc)
            if not self.audit_path.exists():
                return {"deleted": 0, "kept": 0, "bytes": 0}
            original = self.audit_path.read_text(encoding="utf-8").splitlines()
            records = []
            for line in original:
                try:
                    record = json.loads(line)
                    timestamp = datetime.fromisoformat(record["timestamp"])
                    if timestamp.tzinfo is None:
                        timestamp = timestamp.replace(tzinfo=timezone.utc)
                except (KeyError, TypeError, ValueError, json.JSONDecodeError):
                    continue
                if timestamp >= cutoff:
                    records.append((timestamp, line))
            records.sort(key=lambda item: item[0])
            records = records[-self.audit_max_records :]
            total = sum(len(line.encode("utf-8")) + 1 for _, line in records)
            while records and total > self.audit_max_bytes:
                _, line = records.pop(0)
                total -= len(line.encode("utf-8")) + 1
            kept = [line for _, line in records]
            if len(kept) != len(original):
                temporary = self.directory / ".audit.tmp"
                temporary.write_text("".join(line + "\n" for line in kept), encoding="utf-8")
                os.replace(temporary, self.audit_path)
            return {"deleted": len(original) - len(kept), "kept": len(kept), "bytes": total}

    def audit_entries(self, limit=50):
        if not self.audit_path.exists():
            return []
        entries = []
        for line in reversed(self.audit_path.read_text(encoding="utf-8").splitlines()[-limit:]):
            try:
                entries.append(json.loads(line))
            except json.JSONDecodeError:
                continue
        return entries
