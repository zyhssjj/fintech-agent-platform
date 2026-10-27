"""模型配置：LLM / Embedding / Rerank，支持运行时覆盖（密钥 Fernet 加密）。

优先级：runtime/model_config.json 的覆盖 > .env 默认值。
主密钥取自环境变量 MODEL_CONFIG_MASTER_KEY，或本地生成于 runtime/master.key（0600）。
"""

from __future__ import annotations

import json
import os
from pathlib import Path

from dotenv import load_dotenv


PROJECT_DIR = Path(__file__).resolve().parents[1]
RUNTIME_DIR = PROJECT_DIR / "runtime"
CONFIG_PATH = RUNTIME_DIR / "model_config.json"
MASTER_KEY_PATH = RUNTIME_DIR / "master.key"

_ENCRYPTED_PREFIX = "gAAAA"  # Fernet 密文的 base64 前缀特征


def _load_dotenv() -> None:
    load_dotenv(PROJECT_DIR / ".env", override=False)


def _master_key() -> bytes:
    _load_dotenv()
    env_key = os.getenv("MODEL_CONFIG_MASTER_KEY") or os.getenv("LLM_CONFIG_MASTER_KEY", "")
    if env_key:
        return env_key.encode()
    RUNTIME_DIR.mkdir(parents=True, exist_ok=True)
    if MASTER_KEY_PATH.exists():
        return MASTER_KEY_PATH.read_bytes()
    from cryptography.fernet import Fernet
    key = Fernet.generate_key()
    MASTER_KEY_PATH.write_bytes(key)
    os.chmod(MASTER_KEY_PATH, 0o600)
    return key


def _fernet():
    from cryptography.fernet import Fernet
    return Fernet(_master_key())


def encrypt(value: str) -> str:
    return _fernet().encrypt(value.encode("utf-8")).decode("utf-8")


def decrypt(value: str) -> str:
    if not value or not value.startswith(_ENCRYPTED_PREFIX):
        return value
    try:
        return _fernet().decrypt(value.encode("utf-8")).decode("utf-8")
    except Exception:  # noqa: BLE001
        return ""


def load_overrides() -> dict:
    if not CONFIG_PATH.exists():
        return {}
    try:
        return json.loads(CONFIG_PATH.read_text(encoding="utf-8"))
    except (json.JSONDecodeError, OSError):
        return {}


def save_overrides(data: dict) -> None:
    RUNTIME_DIR.mkdir(parents=True, exist_ok=True)
    CONFIG_PATH.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")


def get_config(kind: str) -> dict:
    """kind ∈ {"llm", "embedding", "rerank"}。返回合并后的配置（api_key 已解密）。"""
    _load_dotenv()
    overrides = (load_overrides() or {}).get(kind, {}) or {}
    if kind == "llm":
        env = {
            "base_url": os.getenv("LLM_BASE_URL", ""),
            "api_key": os.getenv("LLM_API_KEY", ""),
            "model": os.getenv("LLM_MODEL", "qwen-max"),
        }
    elif kind == "embedding":
        env = {
            "base_url": os.getenv("EMBEDDING_BASE_URL", ""),
            "api_key": os.getenv("EMBEDDING_API_KEY", ""),
            "model": os.getenv("EMBEDDING_MODEL", "text-embedding-v4"),
            "dimension": int(os.getenv("EMBEDDING_DIMENSION", "1024")),
        }
    else:
        env = {
            "base_url": os.getenv(
                "RERANK_BASE_URL",
                "https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank",
            ),
            "api_key": os.getenv("RERANK_API_KEY") or os.getenv("LLM_API_KEY", ""),
            "model": os.getenv("RERANK_MODEL", "qwen3-vl-rerank"),
        }
    merged = dict(env)
    for key, value in overrides.items():
        if key == "api_key" and isinstance(value, str):
            value = decrypt(value)
        merged[key] = value
    return merged


def save_config(kind: str, values: dict) -> dict:
    data = load_overrides()
    section = dict(data.get(kind, {}) or {})
    for key, value in values.items():
        if key == "api_key" and isinstance(value, str) and value and not value.startswith(_ENCRYPTED_PREFIX):
            value = encrypt(value)
        section[key] = value
    data[kind] = section
    save_overrides(data)
    return get_config(kind)


def masked_config(kind: str) -> dict:
    cfg = get_config(kind)
    key = cfg.get("api_key", "")
    cfg = dict(cfg)
    if key:
        cfg["api_key"] = key[:6] + "***" + key[-4:] if len(key) > 12 else "***"
    else:
        cfg["api_key"] = ""
    return cfg
