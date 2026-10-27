"""Persistent, encrypted registry for OpenAI-compatible LLM endpoints."""

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
MAX_CONTEXT_WINDOW = 1_000_000


def positive_env_int(name, default):
    try:
        return max(1, int(os.getenv(name, default)))
    except (TypeError, ValueError):
        return default


def default_context_window(model_name):
    normalized = (model_name or "").lower().replace("_", "-").replace(" ", "-")
    if "deepseek-v4-flash" in normalized or "deepseekv4flash" in normalized:
        return MAX_CONTEXT_WINDOW
    return 128_000


def utc_now():
    return datetime.now(timezone.utc).isoformat()


class LLMRegistry:
    def __init__(self, runtime_dir, legacy_config=None):
        self.directory = Path(runtime_dir) / "llm_config"
        self.directory.mkdir(parents=True, exist_ok=True)
        self.config_path = self.directory / "models.json"
        self.audit_path = self.directory / "audit.jsonl"
        self.key_path = self.directory / "master.key"
        self.lock = threading.RLock()
        self.audit_retention_days = positive_env_int("LLM_AUDIT_RETENTION_DAYS", 180)
        self.audit_max_records = positive_env_int("LLM_AUDIT_MAX_RECORDS", 100_000)
        self.audit_max_bytes = positive_env_int("LLM_AUDIT_MAX_BYTES", 50 * 1024 * 1024)
        self._last_audit_cleanup = datetime.min.replace(tzinfo=timezone.utc)
        self.cipher = Fernet(self._load_master_key())
        if not self.config_path.exists():
            self._write(
                {
                    "version": 3,
                    "active_model_id": None,
                    "active_ingestion_model_id": None,
                    "models": [],
                }
            )
        if legacy_config:
            self._bootstrap_legacy(legacy_config)
        self._migrate_context_settings()

    def _load_master_key(self):
        environment_key = os.getenv("LLM_CONFIG_MASTER_KEY", "").strip()
        if environment_key:
            encoded = environment_key.encode("utf-8")
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
            raise RuntimeError("LLM model registry is unreadable") from error

    def _write(self, data):
        temporary = self.directory / ".models.tmp"
        temporary.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")
        os.replace(temporary, self.config_path)

    def _encrypt(self, value):
        if not value:
            return None
        return self.cipher.encrypt(value.encode("utf-8")).decode("ascii")

    def _decrypt(self, value):
        if not value:
            return ""
        try:
            return self.cipher.decrypt(value.encode("ascii")).decode("utf-8")
        except InvalidToken as error:
            raise RuntimeError(
                "LLM API key cannot be decrypted; verify LLM_CONFIG_MASTER_KEY"
            ) from error

    def _bootstrap_legacy(self, legacy):
        if not (legacy.get("base_url") and legacy.get("model")):
            return
        with self.lock:
            data = self._load()
            if data["models"]:
                return
            now = utc_now()
            data["models"].append(
                {
                    "id": uuid.uuid4().hex,
                    "name": legacy.get("name") or legacy["model"],
                    "category": "cloud",
                    "provider": "OpenAI-compatible",
                    "base_url": legacy["base_url"].rstrip("/"),
                    "api_key_encrypted": self._encrypt(legacy.get("api_key", "")),
                    "model": legacy["model"],
                    "domains": ["基金层面", "项目层面", "公司层面"],
                    "temperature": 0.1,
                    "max_tokens": 1600,
                    "context_window": default_context_window(legacy["model"]),
                    "timeout": 90,
                    "thinking": legacy.get("thinking", "disabled"),
                    "allow_ingestion": False,
                    "created_at": now,
                    "updated_at": now,
                    "last_test_status": "untested",
                    "last_tested_at": None,
                    "last_test_error": None,
                }
            )
            self._write(data)

    def _migrate_context_settings(self):
        """Add context metadata to registries created before context budgeting."""
        with self.lock:
            data = self._load()
            changed = data.get("version", 1) < 3
            if "active_ingestion_model_id" not in data:
                data["active_ingestion_model_id"] = None
                changed = True
            for model in data.get("models", []):
                if "context_window" not in model:
                    model["context_window"] = default_context_window(model.get("model"))
                    changed = True
                if "allow_ingestion" not in model:
                    model["allow_ingestion"] = False
                    changed = True
            if changed:
                data["version"] = 3
                self._write(data)

    def _normalize(self, payload, existing=None):
        existing = existing or {}
        category = payload.get("category", existing.get("category", "")).strip().lower()
        if category not in VALID_CATEGORIES:
            raise ValueError("模型类型必须是云端模型或本地模型")
        domains = sorted(set(payload.get("domains", existing.get("domains", []))))
        if not domains or not set(domains).issubset(VALID_DOMAINS):
            raise ValueError("请至少选择一个有效知识域")
        if category == "cloud" and "c3" in domains:
            raise ValueError("云端模型禁止配置 C3 知识域权限")
        name = payload.get("name", existing.get("name", "")).strip()
        base_url = payload.get("base_url", existing.get("base_url", "")).strip().rstrip("/")
        model = payload.get("model", existing.get("model", "")).strip()
        provider = payload.get("provider", existing.get("provider", "OpenAI-compatible")).strip()
        if not name or not base_url or not model:
            raise ValueError("模型名称、API 地址和实际模型名称不能为空")
        if not base_url.startswith(("http://", "https://")):
            raise ValueError("API 地址必须以 http:// 或 https:// 开头")

        api_key = payload.get("api_key")
        clear_key = bool(payload.get("clear_api_key"))
        encrypted = existing.get("api_key_encrypted")
        if clear_key:
            encrypted = None
        elif api_key is not None and api_key != "":
            encrypted = self._encrypt(api_key)
        if category == "cloud" and not encrypted:
            raise ValueError("云端模型必须配置 API Key")

        max_tokens = int(payload.get("max_tokens", existing.get("max_tokens", 1600)))
        context_window = int(
            payload.get(
                "context_window",
                existing.get("context_window", default_context_window(model)),
            )
        )
        if not 16_000 <= context_window <= MAX_CONTEXT_WINDOW:
            raise ValueError("上下文上限必须在 16K 到 1M tokens 之间")
        if max_tokens + 8192 >= context_window:
            raise ValueError("最大输出必须小于上下文上限，并至少预留 8192 tokens")

        return {
            "name": name,
            "category": category,
            "provider": provider or "OpenAI-compatible",
            "base_url": base_url,
            "api_key_encrypted": encrypted,
            "model": model,
            "domains": domains,
            "temperature": float(payload.get("temperature", existing.get("temperature", 0.1))),
            "max_tokens": max_tokens,
            "context_window": context_window,
            "timeout": int(payload.get("timeout", existing.get("timeout", 90))),
            "thinking": payload.get("thinking", existing.get("thinking", "disabled")),
            "allow_ingestion": bool(
                payload.get("allow_ingestion", existing.get("allow_ingestion", False))
            ),
        }

    def _public(self, model, active_model_id):
        item = {key: value for key, value in model.items() if key != "api_key_encrypted"}
        secret = self._decrypt(model.get("api_key_encrypted"))
        item["has_api_key"] = bool(secret)
        item["api_key_masked"] = (
            "••••••••{0}".format(secret[-4:]) if secret else "无需密钥"
        )
        item["is_active"] = model["id"] == active_model_id
        item["is_ingestion_active"] = False
        return item

    def list_public(self):
        with self.lock:
            data = self._load()
            items = []
            for model in data["models"]:
                item = self._public(model, data.get("active_model_id"))
                item["is_ingestion_active"] = (
                    model["id"] == data.get("active_ingestion_model_id")
                )
                items.append(item)
            return items

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
                changed_connection = any(
                    normalized[key] != model.get(key)
                    for key in ("base_url", "api_key_encrypted", "model", "thinking")
                )
                updated = {
                    **model,
                    **normalized,
                    "updated_at": utc_now(),
                }
                if changed_connection:
                    updated.update(
                        last_test_status="untested",
                        last_tested_at=None,
                        last_test_error=None,
                    )
                    if data.get("active_model_id") == model_id:
                        data["active_model_id"] = None
                    if data.get("active_ingestion_model_id") == model_id:
                        data["active_ingestion_model_id"] = None
                if not updated.get("allow_ingestion") and data.get("active_ingestion_model_id") == model_id:
                    data["active_ingestion_model_id"] = None
                data["models"][index] = updated
                self._write(data)
                return self._public(updated, data.get("active_model_id"))
        raise KeyError(model_id)

    def delete(self, model_id):
        with self.lock:
            data = self._load()
            before = len(data["models"])
            data["models"] = [model for model in data["models"] if model["id"] != model_id]
            if len(data["models"]) == before:
                raise KeyError(model_id)
            if data.get("active_model_id") == model_id:
                data["active_model_id"] = None
            if data.get("active_ingestion_model_id") == model_id:
                data["active_ingestion_model_id"] = None
            self._write(data)

    def get_private(self, model_id):
        with self.lock:
            data = self._load()
            for model in data["models"]:
                if model["id"] == model_id:
                    return {
                        **model,
                        "api_key": self._decrypt(model.get("api_key_encrypted")),
                    }
        raise KeyError(model_id)

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
                if not success and data.get("active_ingestion_model_id") == model_id:
                    data["active_ingestion_model_id"] = None
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
                raise ValueError("模型必须先通过连接测试才能启用")
            data["active_model_id"] = model_id
            self._write(data)
            return self._public(model, model_id)

    def active_private(self):
        with self.lock:
            data = self._load()
            active_id = data.get("active_model_id")
            if not active_id:
                return None
            model = next((item for item in data["models"] if item["id"] == active_id), None)
            if not model:
                return None
            return {
                **model,
                "api_key": self._decrypt(model.get("api_key_encrypted")),
            }

    def activate_ingestion(self, model_id):
        with self.lock:
            data = self._load()
            model = next((item for item in data["models"] if item["id"] == model_id), None)
            if not model:
                raise KeyError(model_id)
            if model.get("last_test_status") != "success":
                raise ValueError("模型必须先通过连接测试才能用于语义增强入库")
            if not model.get("allow_ingestion"):
                raise ValueError("该模型尚未开启完整原文入库处理权限")
            data["active_ingestion_model_id"] = model_id
            self._write(data)
            public = self._public(model, data.get("active_model_id"))
            public["is_ingestion_active"] = True
            return public

    def active_ingestion_private(self):
        with self.lock:
            data = self._load()
            active_id = data.get("active_ingestion_model_id")
            if not active_id:
                return None
            model = next((item for item in data["models"] if item["id"] == active_id), None)
            if not model or not model.get("allow_ingestion"):
                return None
            return {
                **model,
                "api_key": self._decrypt(model.get("api_key_encrypted")),
            }

    def public_status(self):
        active = self.active_private()
        ingestion_active = self.active_ingestion_private()

        def make_public(model, flag):
            if not model:
                return None
            public = {
                key: value
                for key, value in model.items()
                if key not in {"api_key", "api_key_encrypted"}
            }
            public["has_api_key"] = bool(model.get("api_key"))
            public[flag] = True
            return public

        return {
            "configured": bool(self.list_public()),
            "active": make_public(active, "is_active"),
            "ingestion_active": make_public(
                ingestion_active,
                "is_ingestion_active",
            ),
        }

    def audit_call(
        self,
        model,
        domains,
        chunks,
        success,
        error=None,
        context_tokens=None,
        context_level=None,
        call_type="answer",
        latency_ms=None,
    ):
        record = {
            "timestamp": utc_now(),
            "model_id": model["id"],
            "model_name": model["name"],
            "category": model["category"],
            "domains": sorted(set(domains)),
            "chunks": chunks,
            "context_tokens": context_tokens,
            "context_window": model.get("context_window"),
            "context_level": context_level,
            "call_type": call_type,
            "latency_ms": latency_ms,
            "success": bool(success),
            "error": error,
        }
        line = json.dumps(record, ensure_ascii=False)
        with self.lock:
            with self.audit_path.open("a", encoding="utf-8") as handle:
                handle.write(line + "\n")
            now = datetime.now(timezone.utc)
            if now - self._last_audit_cleanup >= timedelta(hours=1):
                self.cleanup_audit()

    def cleanup_audit(self, retention_days=None, max_records=None, max_bytes=None):
        """Keep model audit history bounded by age, record count, and file size."""
        retention_days = retention_days or self.audit_retention_days
        max_records = max_records or self.audit_max_records
        max_bytes = max_bytes or self.audit_max_bytes
        cutoff = datetime.now(timezone.utc) - timedelta(days=retention_days)
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
            records = records[-max_records:]
            total_bytes = sum(len(line.encode("utf-8")) + 1 for _, line in records)
            while records and total_bytes > max_bytes:
                _, line = records.pop(0)
                total_bytes -= len(line.encode("utf-8")) + 1
            kept_lines = [line for _, line in records]
            deleted = len(original) - len(kept_lines)
            if deleted:
                temporary = self.directory / ".audit.tmp"
                temporary.write_text(
                    "".join("{0}\n".format(line) for line in kept_lines),
                    encoding="utf-8",
                )
                os.replace(temporary, self.audit_path)
            return {"deleted": deleted, "kept": len(kept_lines), "bytes": total_bytes}

    def audit_entries(self, limit=50):
        if not self.audit_path.exists():
            return []
        lines = self.audit_path.read_text(encoding="utf-8").splitlines()
        entries = []
        for line in reversed(lines[-limit:]):
            try:
                entries.append(json.loads(line))
            except json.JSONDecodeError:
                continue
        return entries
