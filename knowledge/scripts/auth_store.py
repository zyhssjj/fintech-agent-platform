"""SQLite-backed users, roles, knowledge domains, and server-side sessions."""

import hashlib
import hmac
import json
import secrets
import sqlite3
import threading
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path


ROLES = {"superadmin", "admin", "user"}
DOMAINS = {"基金层面", "项目层面", "公司层面"}
ALL_DOMAINS = ["基金层面", "项目层面", "公司层面"]
NORMAL_USER_DOMAINS = set(ALL_DOMAINS)
API_CLIENT_DOMAINS = set(ALL_DOMAINS)
ROLE_LABELS = {
    "superadmin": "超级管理员",
    "admin": "管理员",
    "user": "普通用户",
}


def utc_now():
    return datetime.now(timezone.utc)


def utc_iso(value=None):
    return (value or utc_now()).isoformat()


def normalize_username(value):
    return value.strip().casefold()


def hash_password(password):
    if len(password) < 8:
        raise ValueError("密码至少需要8个字符")
    salt = secrets.token_bytes(16)
    iterations = 600_000
    digest = hashlib.pbkdf2_hmac(
        "sha256",
        password.encode("utf-8"),
        salt,
        iterations,
        dklen=32,
    )
    return "pbkdf2_sha256${0}${1}${2}".format(
        iterations,
        salt.hex(),
        digest.hex(),
    )


def verify_password(password, encoded):
    try:
        parts = encoded.split("$")
        algorithm = parts[0]
        if algorithm == "pbkdf2_sha256":
            _, iterations, salt, expected = parts
            actual = hashlib.pbkdf2_hmac(
                "sha256",
                password.encode("utf-8"),
                bytes.fromhex(salt),
                int(iterations),
                dklen=len(bytes.fromhex(expected)),
            )
        elif algorithm == "scrypt":
            _, cost, r_value, p_value, salt, expected = parts
            actual = hashlib.scrypt(
                password.encode("utf-8"),
                salt=bytes.fromhex(salt),
                n=int(cost),
                r=int(r_value),
                p=int(p_value),
                dklen=len(bytes.fromhex(expected)),
            )
        else:
            return False
        return hmac.compare_digest(actual.hex(), expected)
    except (AttributeError, ValueError, TypeError):
        return False


class AuthStore:
    def __init__(self, runtime_dir, session_hours=12):
        self.directory = Path(runtime_dir) / "auth"
        self.directory.mkdir(parents=True, exist_ok=True)
        self.database_path = self.directory / "users.db"
        self.session_hours = session_hours
        self.lock = threading.RLock()
        self._initialize()

    def _connect(self):
        connection = sqlite3.connect(
            self.database_path,
            timeout=30,
            isolation_level=None,
        )
        connection.row_factory = sqlite3.Row
        connection.execute("PRAGMA foreign_keys = ON")
        connection.execute("PRAGMA journal_mode = WAL")
        return connection

    def _initialize(self):
        with self.lock, self._connect() as connection:
            connection.executescript(
                """
                CREATE TABLE IF NOT EXISTS users (
                    id TEXT PRIMARY KEY,
                    name TEXT NOT NULL,
                    position TEXT NOT NULL,
                    username TEXT NOT NULL,
                    username_normalized TEXT NOT NULL UNIQUE,
                    password_hash TEXT NOT NULL,
                    role TEXT NOT NULL CHECK (role IN ('superadmin', 'admin', 'user')),
                    domains_json TEXT NOT NULL,
                    must_change_password INTEGER NOT NULL DEFAULT 1,
                    created_at TEXT NOT NULL,
                    updated_at TEXT NOT NULL
                );

                CREATE TABLE IF NOT EXISTS user_credential_bindings (
                    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                    credential_id TEXT NOT NULL,
                    created_at TEXT NOT NULL,
                    PRIMARY KEY (user_id, credential_id)
                );

                CREATE INDEX IF NOT EXISTS user_credential_bindings_credential
                ON user_credential_bindings(credential_id);

                CREATE UNIQUE INDEX IF NOT EXISTS one_superadmin
                ON users(role) WHERE role = 'superadmin';

                CREATE TABLE IF NOT EXISTS sessions (
                    token_hash TEXT PRIMARY KEY,
                    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                    csrf_token TEXT NOT NULL,
                    created_at TEXT NOT NULL,
                    expires_at TEXT NOT NULL
                );

                CREATE TABLE IF NOT EXISTS auth_audit (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    actor_user_id TEXT,
                    action TEXT NOT NULL,
                    target_user_id TEXT,
                    detail TEXT,
                    created_at TEXT NOT NULL
                );

                CREATE TABLE IF NOT EXISTS api_clients (
                    id TEXT PRIMARY KEY,
                    name TEXT NOT NULL,
                    purpose TEXT NOT NULL,
                    key_prefix TEXT NOT NULL UNIQUE,
                    token_hash TEXT NOT NULL UNIQUE,
                    domains_json TEXT NOT NULL,
                    allow_retrieve INTEGER NOT NULL DEFAULT 1,
                    allow_answer INTEGER NOT NULL DEFAULT 0,
                    rate_limit_per_minute INTEGER NOT NULL DEFAULT 60,
                    enabled INTEGER NOT NULL DEFAULT 1,
                    expires_at TEXT,
                    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
                    created_at TEXT NOT NULL,
                    updated_at TEXT NOT NULL,
                    last_used_at TEXT
                );

                CREATE TABLE IF NOT EXISTS api_client_audit (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    client_id TEXT,
                    endpoint TEXT NOT NULL,
                    query_hash TEXT,
                    query_length INTEGER NOT NULL DEFAULT 0,
                    domains_json TEXT NOT NULL,
                    result_count INTEGER NOT NULL DEFAULT 0,
                    latency_ms INTEGER NOT NULL DEFAULT 0,
                    status_code INTEGER NOT NULL,
                    success INTEGER NOT NULL,
                    detail TEXT,
                    created_at TEXT NOT NULL
                );

                CREATE INDEX IF NOT EXISTS api_client_audit_created_at
                ON api_client_audit(created_at);
                """
            )
            columns = {
                row["name"]
                for row in connection.execute("PRAGMA table_info(users)").fetchall()
            }
            if "api_key_sha256" not in columns:
                connection.execute("ALTER TABLE users ADD COLUMN api_key_sha256 TEXT")
            if "api_key_prefix" not in columns:
                connection.execute("ALTER TABLE users ADD COLUMN api_key_prefix TEXT")
            connection.execute(
                "CREATE UNIQUE INDEX IF NOT EXISTS users_api_key_sha256 ON users(api_key_sha256) WHERE api_key_sha256 IS NOT NULL"
            )
        try:
            self.database_path.chmod(0o600)
        except OSError:
            pass

    def _public_user(self, row):
        if not row:
            return None
        domains = sorted(set(json.loads(row["domains_json"])) & DOMAINS)
        if row["role"] in {"superadmin", "admin"}:
            domains = list(ALL_DOMAINS)
        elif row["role"] == "user":
            domains = sorted(set(domains) & NORMAL_USER_DOMAINS)
        return {
            "id": row["id"],
            "name": row["name"],
            "position": row["position"],
            "username": row["username"],
            "role": row["role"],
            "role_label": ROLE_LABELS[row["role"]],
            "domains": domains,
            "must_change_password": bool(row["must_change_password"]),
            "api_key_prefix": row["api_key_prefix"] if "api_key_prefix" in row.keys() else None,
            "created_at": row["created_at"],
            "updated_at": row["updated_at"],
        }

    @staticmethod
    def _credential_ids(connection, user_id):
        return [
            row["credential_id"]
            for row in connection.execute(
                "SELECT credential_id FROM user_credential_bindings WHERE user_id = ? ORDER BY rowid",
                (user_id,),
            ).fetchall()
        ]

    def _with_credential_ids(self, connection, user):
        if user:
            user["credential_ids"] = self._credential_ids(connection, user["id"])
        return user

    def has_users(self):
        with self._connect() as connection:
            return connection.execute("SELECT 1 FROM users LIMIT 1").fetchone() is not None

    def has_superadmin(self):
        with self._connect() as connection:
            return (
                connection.execute(
                    "SELECT 1 FROM users WHERE role = 'superadmin' LIMIT 1"
                ).fetchone()
                is not None
            )

    def get_user(self, user_id):
        with self._connect() as connection:
            row = connection.execute(
                "SELECT * FROM users WHERE id = ?",
                (user_id,),
            ).fetchone()
            return self._with_credential_ids(connection, self._public_user(row))

    def get_user_by_username(self, username):
        with self._connect() as connection:
            row = connection.execute(
                "SELECT * FROM users WHERE username_normalized = ?",
                (normalize_username(username),),
            ).fetchone()
        return row

    def list_users(self):
        with self._connect() as connection:
            rows = connection.execute(
                """
                SELECT * FROM users
                ORDER BY
                    CASE role
                        WHEN 'superadmin' THEN 0
                        WHEN 'admin' THEN 1
                        ELSE 2
                    END,
                    created_at
                """
            ).fetchall()
            return [self._with_credential_ids(connection, self._public_user(row)) for row in rows]

    def set_user_credentials(self, user_id, credential_ids, actor_user_id=None):
        if not self.get_user(user_id):
            raise KeyError(user_id)
        selected = list(dict.fromkeys(str(item).strip() for item in (credential_ids or []) if str(item).strip()))
        with self.lock, self._connect() as connection:
            connection.execute("BEGIN IMMEDIATE")
            try:
                connection.execute("DELETE FROM user_credential_bindings WHERE user_id = ?", (user_id,))
                now = utc_iso()
                connection.executemany(
                    "INSERT INTO user_credential_bindings (user_id, credential_id, created_at) VALUES (?, ?, ?)",
                    [(user_id, credential_id, now) for credential_id in selected],
                )
                self._audit(
                    connection,
                    actor_user_id,
                    "bind_user_credentials",
                    user_id,
                    {"credential_ids": selected},
                )
                connection.execute("COMMIT")
            except Exception:
                connection.execute("ROLLBACK")
                raise
        return self.get_user(user_id)

    def remove_credential_bindings(self, credential_id):
        with self.lock, self._connect() as connection:
            return connection.execute(
                "DELETE FROM user_credential_bindings WHERE credential_id = ?",
                (credential_id,),
            ).rowcount

    def rotate_user_api_key(self, user_id, actor_user_id=None):
        if not self.get_user(user_id):
            raise KeyError(user_id)
        key = "wk_user_" + secrets.token_hex(24)
        digest = hashlib.sha256(key.encode("utf-8")).hexdigest()
        prefix = key[:16]
        with self.lock, self._connect() as connection:
            connection.execute(
                "UPDATE users SET api_key_sha256 = ?, api_key_prefix = ?, updated_at = ? WHERE id = ?",
                (digest, prefix, utc_iso(), user_id),
            )
            self._audit(connection, actor_user_id, "rotate_user_api_key", user_id, {})
        return self.get_user(user_id), key

    def authenticate_user_api_key(self, key):
        if not key or not key.startswith("wk_user_"):
            return None
        digest = hashlib.sha256(key.encode("utf-8")).hexdigest()
        with self._connect() as connection:
            row = connection.execute(
                "SELECT * FROM users WHERE api_key_sha256 = ?",
                (digest,),
            ).fetchone()
            return self._with_credential_ids(connection, self._public_user(row))

    def _public_api_client(self, row):
        if not row:
            return None
        domains = sorted(set(json.loads(row["domains_json"])) & API_CLIENT_DOMAINS)
        expires_at = row["expires_at"]
        expired = bool(
            expires_at and datetime.fromisoformat(expires_at) <= utc_now()
        )
        capabilities = []
        if row["allow_retrieve"]:
            capabilities.append("retrieve")
        if row["allow_answer"]:
            capabilities.append("answer")
        return {
            "id": row["id"],
            "name": row["name"],
            "purpose": row["purpose"],
            "key_prefix": row["key_prefix"],
            "domains": domains,
            "capabilities": capabilities,
            "allow_retrieve": bool(row["allow_retrieve"]),
            "allow_answer": bool(row["allow_answer"]),
            "rate_limit_per_minute": row["rate_limit_per_minute"],
            "enabled": bool(row["enabled"]) and not expired,
            "configured_enabled": bool(row["enabled"]),
            "expired": expired,
            "expires_at": expires_at,
            "created_at": row["created_at"],
            "updated_at": row["updated_at"],
            "last_used_at": row["last_used_at"],
        }

    def list_api_clients(self):
        with self._connect() as connection:
            rows = connection.execute(
                "SELECT * FROM api_clients ORDER BY created_at DESC"
            ).fetchall()
        return [self._public_api_client(row) for row in rows]

    def get_api_client(self, client_id):
        with self._connect() as connection:
            row = connection.execute(
                "SELECT * FROM api_clients WHERE id = ?",
                (client_id,),
            ).fetchone()
        return self._public_api_client(row)

    def create_api_client(
        self,
        name,
        purpose,
        domains,
        *,
        allow_retrieve=True,
        allow_answer=False,
        rate_limit_per_minute=60,
        enabled=True,
        expires_at=None,
        actor_user_id=None,
    ):
        name = name.strip()
        purpose = purpose.strip()
        selected_domains = sorted(set(domains or []) & API_CLIENT_DOMAINS)
        if not name:
            raise ValueError("工具名称不能为空")
        if not purpose:
            raise ValueError("用途说明不能为空")
        if not selected_domains:
            raise ValueError("至少选择一个可访问知识域")
        if not allow_retrieve and not allow_answer:
            raise ValueError("至少启用一种接口能力")
        rate_limit_per_minute = max(1, min(600, int(rate_limit_per_minute)))
        if expires_at:
            parsed_expiry = datetime.fromisoformat(expires_at)
            if parsed_expiry <= utc_now():
                raise ValueError("凭证过期时间必须晚于当前时间")
        now = utc_iso()
        client_id = uuid.uuid4().hex
        token = "lem_kb_{0}".format(secrets.token_urlsafe(36))
        key_prefix = token[:18]
        token_hash = hashlib.sha256(token.encode("utf-8")).hexdigest()
        with self.lock, self._connect() as connection:
            connection.execute("BEGIN IMMEDIATE")
            try:
                connection.execute(
                    """
                    INSERT INTO api_clients (
                        id, name, purpose, key_prefix, token_hash, domains_json,
                        allow_retrieve, allow_answer, rate_limit_per_minute,
                        enabled, expires_at, created_by_user_id,
                        created_at, updated_at
                    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                    """,
                    (
                        client_id,
                        name,
                        purpose,
                        key_prefix,
                        token_hash,
                        json.dumps(selected_domains),
                        int(bool(allow_retrieve)),
                        int(bool(allow_answer)),
                        rate_limit_per_minute,
                        int(bool(enabled)),
                        expires_at,
                        actor_user_id,
                        now,
                        now,
                    ),
                )
                self._audit(
                    connection,
                    actor_user_id,
                    "create_api_client",
                    None,
                    {
                        "api_client_id": client_id,
                        "name": name,
                        "domains": selected_domains,
                        "allow_retrieve": bool(allow_retrieve),
                        "allow_answer": bool(allow_answer),
                    },
                )
                connection.execute("COMMIT")
            except Exception:
                connection.execute("ROLLBACK")
                raise
        return self.get_api_client(client_id), token

    def update_api_client(
        self,
        client_id,
        *,
        name,
        purpose,
        domains,
        allow_retrieve,
        allow_answer,
        rate_limit_per_minute,
        enabled,
        expires_at,
        actor_user_id=None,
    ):
        if not self.get_api_client(client_id):
            raise KeyError(client_id)
        name = name.strip()
        purpose = purpose.strip()
        selected_domains = sorted(set(domains or []) & API_CLIENT_DOMAINS)
        if not name or not purpose:
            raise ValueError("工具名称和用途说明不能为空")
        if not selected_domains:
            raise ValueError("至少选择一个可访问知识域")
        if not allow_retrieve and not allow_answer:
            raise ValueError("至少启用一种接口能力")
        if expires_at and datetime.fromisoformat(expires_at) <= utc_now():
            raise ValueError("凭证过期时间必须晚于当前时间")
        rate_limit_per_minute = max(1, min(600, int(rate_limit_per_minute)))
        with self.lock, self._connect() as connection:
            connection.execute("BEGIN IMMEDIATE")
            try:
                connection.execute(
                    """
                    UPDATE api_clients
                    SET name = ?, purpose = ?, domains_json = ?,
                        allow_retrieve = ?, allow_answer = ?,
                        rate_limit_per_minute = ?, enabled = ?,
                        expires_at = ?, updated_at = ?
                    WHERE id = ?
                    """,
                    (
                        name,
                        purpose,
                        json.dumps(selected_domains),
                        int(bool(allow_retrieve)),
                        int(bool(allow_answer)),
                        rate_limit_per_minute,
                        int(bool(enabled)),
                        expires_at,
                        utc_iso(),
                        client_id,
                    ),
                )
                self._audit(
                    connection,
                    actor_user_id,
                    "update_api_client",
                    None,
                    {
                        "api_client_id": client_id,
                        "domains": selected_domains,
                        "allow_retrieve": bool(allow_retrieve),
                        "allow_answer": bool(allow_answer),
                        "enabled": bool(enabled),
                    },
                )
                connection.execute("COMMIT")
            except Exception:
                connection.execute("ROLLBACK")
                raise
        return self.get_api_client(client_id)

    def rotate_api_client_key(self, client_id, actor_user_id=None):
        current = self.get_api_client(client_id)
        if not current:
            raise KeyError(client_id)
        token = "lem_kb_{0}".format(secrets.token_urlsafe(36))
        key_prefix = token[:18]
        token_hash = hashlib.sha256(token.encode("utf-8")).hexdigest()
        with self.lock, self._connect() as connection:
            connection.execute(
                """
                UPDATE api_clients
                SET key_prefix = ?, token_hash = ?, updated_at = ?
                WHERE id = ?
                """,
                (key_prefix, token_hash, utc_iso(), client_id),
            )
            self._audit(
                connection,
                actor_user_id,
                "rotate_api_client_key",
                None,
                {"api_client_id": client_id},
            )
        return self.get_api_client(client_id), token

    def revoke_api_client(self, client_id, actor_user_id=None):
        current = self.get_api_client(client_id)
        if not current:
            raise KeyError(client_id)
        with self.lock, self._connect() as connection:
            connection.execute(
                """
                UPDATE api_clients
                SET enabled = 0, updated_at = ?
                WHERE id = ?
                """,
                (utc_iso(), client_id),
            )
            self._audit(
                connection,
                actor_user_id,
                "revoke_api_client",
                None,
                {"api_client_id": client_id, "name": current["name"]},
            )
        return self.get_api_client(client_id)

    def authenticate_api_client(self, token):
        if not token or not token.startswith("lem_kb_"):
            return None
        token_hash = hashlib.sha256(token.encode("utf-8")).hexdigest()
        with self.lock, self._connect() as connection:
            row = connection.execute(
                "SELECT * FROM api_clients WHERE token_hash = ?",
                (token_hash,),
            ).fetchone()
            client = self._public_api_client(row)
            if not client or not client["enabled"]:
                return None
            connection.execute(
                "UPDATE api_clients SET last_used_at = ? WHERE id = ?",
                (utc_iso(), client["id"]),
            )
        client["last_used_at"] = utc_iso()
        return client

    def record_api_call(
        self,
        client_id,
        endpoint,
        *,
        query_hash=None,
        query_length=0,
        domains=None,
        result_count=0,
        latency_ms=0,
        status_code=200,
        success=True,
        detail=None,
    ):
        with self.lock, self._connect() as connection:
            connection.execute(
                """
                INSERT INTO api_client_audit (
                    client_id, endpoint, query_hash, query_length,
                    domains_json, result_count, latency_ms,
                    status_code, success, detail, created_at
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    client_id,
                    endpoint,
                    query_hash,
                    int(query_length),
                    json.dumps(sorted(set(domains or []) & DOMAINS)),
                    int(result_count),
                    int(latency_ms),
                    int(status_code),
                    int(bool(success)),
                    json.dumps(detail or {}, ensure_ascii=False),
                    utc_iso(),
                ),
            )

    def recent_api_calls(self, limit=50):
        limit = max(1, min(200, int(limit)))
        with self._connect() as connection:
            rows = connection.execute(
                """
                SELECT audit.*, clients.name AS client_name
                FROM api_client_audit AS audit
                LEFT JOIN api_clients AS clients ON clients.id = audit.client_id
                ORDER BY audit.id DESC LIMIT ?
                """,
                (limit,),
            ).fetchall()
        return [
            {
                "id": row["id"],
                "client_id": row["client_id"],
                "client_name": row["client_name"] or "已撤销凭证",
                "endpoint": row["endpoint"],
                "query_hash": row["query_hash"],
                "query_length": row["query_length"],
                "domains": json.loads(row["domains_json"]),
                "result_count": row["result_count"],
                "latency_ms": row["latency_ms"],
                "status_code": row["status_code"],
                "success": bool(row["success"]),
                "created_at": row["created_at"],
            }
            for row in rows
        ]

    def create_user(
        self,
        name,
        position,
        username,
        password,
        role="user",
        domains=None,
        must_change_password=True,
        actor_user_id=None,
    ):
        if role not in ROLES:
            raise ValueError("用户身份无效")
        if not name.strip() or not position.strip():
            raise ValueError("名字和职位不能为空")
        normalized = normalize_username(username)
        if not normalized:
            raise ValueError("账号不能为空")
        selected_domains = sorted(set(domains or []) & DOMAINS)
        if role in {"superadmin", "admin"}:
            selected_domains = list(ALL_DOMAINS)
        else:
            selected_domains = sorted(set(selected_domains) & NORMAL_USER_DOMAINS)
        now = utc_iso()
        user_id = uuid.uuid4().hex
        with self.lock, self._connect() as connection:
            try:
                connection.execute("BEGIN IMMEDIATE")
                connection.execute(
                    """
                    INSERT INTO users (
                        id, name, position, username, username_normalized,
                        password_hash, role, domains_json,
                        must_change_password, created_at, updated_at
                    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                    """,
                    (
                        user_id,
                        name.strip(),
                        position.strip(),
                        username.strip(),
                        normalized,
                        hash_password(password),
                        role,
                        json.dumps(selected_domains),
                        int(bool(must_change_password)),
                        now,
                        now,
                    ),
                )
                self._audit(
                    connection,
                    actor_user_id,
                    "create_user",
                    user_id,
                    {"role": role, "domains": selected_domains},
                )
                connection.execute("COMMIT")
            except sqlite3.IntegrityError as error:
                connection.execute("ROLLBACK")
                if "one_superadmin" in str(error) or role == "superadmin":
                    raise ValueError("系统只能存在一个超级管理员") from error
                raise ValueError("该账号已存在") from error
            except Exception:
                connection.execute("ROLLBACK")
                raise
        return self.get_user(user_id)

    def authenticate(self, username, password):
        row = self.get_user_by_username(username)
        if not row or not verify_password(password, row["password_hash"]):
            return None
        return self._public_user(row)

    def create_session(self, user_id):
        token = secrets.token_urlsafe(48)
        token_hash = hashlib.sha256(token.encode("utf-8")).hexdigest()
        csrf_token = secrets.token_urlsafe(32)
        created = utc_now()
        expires = created + timedelta(hours=self.session_hours)
        with self.lock, self._connect() as connection:
            connection.execute("DELETE FROM sessions WHERE expires_at <= ?", (utc_iso(),))
            connection.execute(
                """
                INSERT INTO sessions (
                    token_hash, user_id, csrf_token, created_at, expires_at
                ) VALUES (?, ?, ?, ?, ?)
                """,
                (
                    token_hash,
                    user_id,
                    csrf_token,
                    utc_iso(created),
                    utc_iso(expires),
                ),
            )
        return token, csrf_token, utc_iso(expires)

    def session_user(self, token):
        if not token:
            return None, None
        token_hash = hashlib.sha256(token.encode("utf-8")).hexdigest()
        with self._connect() as connection:
            row = connection.execute(
                """
                SELECT users.*, sessions.csrf_token, sessions.expires_at
                FROM sessions
                JOIN users ON users.id = sessions.user_id
                WHERE sessions.token_hash = ?
                """,
                (token_hash,),
            ).fetchone()
            if not row:
                return None, None
            if datetime.fromisoformat(row["expires_at"]) <= utc_now():
                connection.execute(
                    "DELETE FROM sessions WHERE token_hash = ?",
                    (token_hash,),
                )
                return None, None
        return self._public_user(row), row["csrf_token"]

    def delete_session(self, token):
        if not token:
            return
        token_hash = hashlib.sha256(token.encode("utf-8")).hexdigest()
        with self.lock, self._connect() as connection:
            connection.execute(
                "DELETE FROM sessions WHERE token_hash = ?",
                (token_hash,),
            )

    def delete_user_sessions(self, user_id):
        with self.lock, self._connect() as connection:
            connection.execute("DELETE FROM sessions WHERE user_id = ?", (user_id,))

    def revoke_c3_from_normal_users(self):
        """Apply the reserved-C3 policy to accounts created before the policy."""
        changed = 0
        with self.lock, self._connect() as connection:
            rows = connection.execute(
                "SELECT id, domains_json FROM users WHERE role = 'user'"
            ).fetchall()
            for row in rows:
                previous = sorted(set(json.loads(row["domains_json"])) & DOMAINS)
                domains = sorted(set(previous) & NORMAL_USER_DOMAINS)
                if domains == previous:
                    continue
                connection.execute(
                    "UPDATE users SET domains_json = ?, updated_at = ? WHERE id = ?",
                    (json.dumps(domains), utc_iso(), row["id"]),
                )
                self._audit(
                    connection,
                    None,
                    "revoke_reserved_c3",
                    row["id"],
                    {"domains": domains},
                )
                changed += 1
        return changed

    def record_event(self, actor_user_id, action, detail=None):
        with self.lock, self._connect() as connection:
            self._audit(connection, actor_user_id, action, None, detail or {})

    def cleanup_audit(self, retention_days=365, max_records=100_000):
        """Bound the SQLite permission audit table without touching users or sessions."""
        cutoff = utc_iso(utc_now() - timedelta(days=max(1, int(retention_days))))
        max_records = max(1, int(max_records))
        with self.lock, self._connect() as connection:
            connection.execute("BEGIN IMMEDIATE")
            removed_by_age = connection.execute(
                "DELETE FROM auth_audit WHERE created_at < ?", (cutoff,)
            ).rowcount
            removed_by_count = connection.execute(
                """
                DELETE FROM auth_audit
                WHERE id NOT IN (
                    SELECT id FROM auth_audit ORDER BY id DESC LIMIT ?
                )
                """,
                (max_records,),
            ).rowcount
            api_removed_by_age = connection.execute(
                "DELETE FROM api_client_audit WHERE created_at < ?", (cutoff,)
            ).rowcount
            api_removed_by_count = connection.execute(
                """
                DELETE FROM api_client_audit
                WHERE id NOT IN (
                    SELECT id FROM api_client_audit ORDER BY id DESC LIMIT ?
                )
                """,
                (max_records,),
            ).rowcount
            connection.execute("COMMIT")
        return {
            "deleted": (
                max(0, removed_by_age)
                + max(0, removed_by_count)
                + max(0, api_removed_by_age)
                + max(0, api_removed_by_count)
            ),
            "retention_days": retention_days,
            "max_records": max_records,
        }

    def update_user(
        self,
        user_id,
        *,
        name,
        position,
        username,
        role,
        domains,
        password=None,
        must_change_password=None,
        actor_user_id=None,
    ):
        current = self.get_user(user_id)
        if not current:
            raise KeyError(user_id)
        if not name.strip() or not position.strip():
            raise ValueError("名字和职位不能为空")
        if current["role"] == "superadmin" and role != "superadmin":
            raise ValueError("不能取消唯一超级管理员身份")
        if role == "superadmin" and current["role"] != "superadmin":
            raise ValueError("系统只能存在一个超级管理员")
        selected_domains = sorted(set(domains or []) & DOMAINS)
        if role in {"superadmin", "admin"}:
            selected_domains = list(ALL_DOMAINS)
        else:
            selected_domains = sorted(set(selected_domains) & NORMAL_USER_DOMAINS)
        normalized = normalize_username(username)
        fields = [
            "name = ?",
            "position = ?",
            "username = ?",
            "username_normalized = ?",
            "role = ?",
            "domains_json = ?",
            "updated_at = ?",
        ]
        values = [
            name.strip(),
            position.strip(),
            username.strip(),
            normalized,
            role,
            json.dumps(selected_domains),
            utc_iso(),
        ]
        password_changed = bool(password)
        if password_changed:
            fields.extend(["password_hash = ?", "must_change_password = ?"])
            values.extend(
                [
                    hash_password(password),
                    int(True if must_change_password is None else must_change_password),
                ]
            )
        elif must_change_password is not None:
            fields.append("must_change_password = ?")
            values.append(int(bool(must_change_password)))
        values.append(user_id)
        with self.lock, self._connect() as connection:
            try:
                connection.execute("BEGIN IMMEDIATE")
                connection.execute(
                    "UPDATE users SET {0} WHERE id = ?".format(", ".join(fields)),
                    values,
                )
                if password_changed:
                    connection.execute(
                        "DELETE FROM sessions WHERE user_id = ?",
                        (user_id,),
                    )
                self._audit(
                    connection,
                    actor_user_id,
                    "update_user",
                    user_id,
                    {"role": role, "domains": selected_domains},
                )
                connection.execute("COMMIT")
            except sqlite3.IntegrityError as error:
                connection.execute("ROLLBACK")
                raise ValueError("该账号已存在") from error
            except Exception:
                connection.execute("ROLLBACK")
                raise
        return self.get_user(user_id)

    def delete_user(self, user_id, actor_user_id=None):
        current = self.get_user(user_id)
        if not current:
            raise KeyError(user_id)
        if current["role"] == "superadmin":
            raise ValueError("不能删除唯一超级管理员")
        with self.lock, self._connect() as connection:
            connection.execute("BEGIN IMMEDIATE")
            self._audit(
                connection,
                actor_user_id,
                "delete_user",
                user_id,
                {
                    "name": current["name"],
                    "username": current["username"],
                    "role": current["role"],
                },
            )
            connection.execute("DELETE FROM users WHERE id = ?", (user_id,))
            connection.execute("COMMIT")

    def verify_user_password(self, user_id, password):
        with self._connect() as connection:
            row = connection.execute(
                "SELECT password_hash FROM users WHERE id = ?",
                (user_id,),
            ).fetchone()
        return bool(row) and verify_password(password, row["password_hash"])

    def _audit(self, connection, actor_user_id, action, target_user_id, detail):
        connection.execute(
            """
            INSERT INTO auth_audit (
                actor_user_id, action, target_user_id, detail, created_at
            ) VALUES (?, ?, ?, ?, ?)
            """,
            (
                actor_user_id,
                action,
                target_user_id,
                json.dumps(detail, ensure_ascii=False),
                utc_iso(),
            ),
        )
