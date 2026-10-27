CREATE TABLE IF NOT EXISTS com_operation_log (
  id BIGSERIAL PRIMARY KEY,
  request_id VARCHAR(64) NOT NULL DEFAULT '',
  user_id VARCHAR(64) NOT NULL DEFAULT '',
  username VARCHAR(64) NOT NULL DEFAULT '',
  method VARCHAR(16) NOT NULL,
  path VARCHAR(512) NOT NULL,
  client_ip VARCHAR(64) NOT NULL DEFAULT '',
  user_agent VARCHAR(512) NOT NULL DEFAULT '',
  status INT NOT NULL,
  latency_ms BIGINT NOT NULL DEFAULT 0,
  params JSONB NOT NULL DEFAULT '{}'::jsonb,
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_com_operation_log_request_id ON com_operation_log(request_id);
CREATE INDEX IF NOT EXISTS idx_com_operation_log_user_id ON com_operation_log(user_id);
CREATE INDEX IF NOT EXISTS idx_com_operation_log_username ON com_operation_log(username);
CREATE INDEX IF NOT EXISTS idx_com_operation_log_path ON com_operation_log(path);
CREATE INDEX IF NOT EXISTS idx_com_operation_log_client_ip ON com_operation_log(client_ip);
CREATE INDEX IF NOT EXISTS idx_com_operation_log_status ON com_operation_log(status);
CREATE INDEX IF NOT EXISTS idx_com_operation_log_is_deleted ON com_operation_log(is_deleted);
