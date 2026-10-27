CREATE TABLE IF NOT EXISTS com_async_task (
  id BIGSERIAL PRIMARY KEY,
  task_id VARCHAR(64) NOT NULL,
  task_type VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  progress INT NOT NULL DEFAULT 0,
  input_params JSONB NOT NULL DEFAULT '{}'::jsonb,
  output_result JSONB NOT NULL DEFAULT '{}'::jsonb,
  error_message TEXT NOT NULL DEFAULT '',
  retry_count INT NOT NULL DEFAULT 0,
  retry_limit INT NOT NULL DEFAULT 3,
  started_at TIMESTAMPTZ NULL,
  finished_at TIMESTAMPTZ NULL,
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  CONSTRAINT uk_com_async_task_task_id UNIQUE (task_id)
);

CREATE INDEX IF NOT EXISTS idx_com_async_task_type ON com_async_task(task_type);
CREATE INDEX IF NOT EXISTS idx_com_async_task_status ON com_async_task(status);
CREATE INDEX IF NOT EXISTS idx_com_async_task_is_deleted ON com_async_task(is_deleted);
CREATE INDEX IF NOT EXISTS idx_com_async_task_input_params ON com_async_task USING GIN (input_params);
CREATE INDEX IF NOT EXISTS idx_com_async_task_output_result ON com_async_task USING GIN (output_result);
