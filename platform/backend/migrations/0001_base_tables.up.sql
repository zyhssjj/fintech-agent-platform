CREATE TABLE IF NOT EXISTS sys_department (
  id BIGSERIAL PRIMARY KEY,
  parent_id BIGINT NULL,
  dept_code VARCHAR(64) NOT NULL,
  dept_name VARCHAR(128) NOT NULL,
  leader_user_id BIGINT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  CONSTRAINT uk_sys_department_code UNIQUE (dept_code),
  CONSTRAINT fk_sys_department_parent FOREIGN KEY (parent_id)
    REFERENCES sys_department(id) ON UPDATE CASCADE ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS sys_user (
  id BIGSERIAL PRIMARY KEY,
  username VARCHAR(64) NOT NULL,
  password VARCHAR(255) NOT NULL,
  real_name VARCHAR(64) NOT NULL DEFAULT '',
  email VARCHAR(128) NOT NULL DEFAULT '',
  phone VARCHAR(32) NOT NULL DEFAULT '',
  avatar VARCHAR(512) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  department_id BIGINT NULL,
  last_login_time TIMESTAMPTZ NULL,
  last_login_ip VARCHAR(64) NOT NULL DEFAULT '',
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  CONSTRAINT uk_sys_user_username UNIQUE (username),
  CONSTRAINT fk_sys_user_department FOREIGN KEY (department_id)
    REFERENCES sys_department(id) ON UPDATE CASCADE ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS sys_role (
  id BIGSERIAL PRIMARY KEY,
  role_code VARCHAR(64) NOT NULL,
  role_name VARCHAR(64) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  CONSTRAINT uk_sys_role_code UNIQUE (role_code)
);

CREATE TABLE IF NOT EXISTS sys_permission (
  id BIGSERIAL PRIMARY KEY,
  perm_code VARCHAR(128) NOT NULL,
  perm_name VARCHAR(128) NOT NULL,
  resource VARCHAR(64) NOT NULL,
  action VARCHAR(64) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  CONSTRAINT uk_sys_permission_code UNIQUE (perm_code)
);

CREATE TABLE IF NOT EXISTS sys_user_role (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  role_id BIGINT NOT NULL,
  fund_scope JSONB NOT NULL DEFAULT '{}'::jsonb,
  project_scope JSONB NOT NULL DEFAULT '{}'::jsonb,
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  CONSTRAINT uk_sys_user_role UNIQUE (user_id, role_id),
  CONSTRAINT fk_sys_user_role_user FOREIGN KEY (user_id)
    REFERENCES sys_user(id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT fk_sys_user_role_role FOREIGN KEY (role_id)
    REFERENCES sys_role(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sys_role_perm (
  id BIGSERIAL PRIMARY KEY,
  role_id BIGINT NOT NULL,
  permission_id BIGINT NOT NULL,
  create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  CONSTRAINT uk_sys_role_perm UNIQUE (role_id, permission_id),
  CONSTRAINT fk_sys_role_perm_role FOREIGN KEY (role_id)
    REFERENCES sys_role(id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT fk_sys_role_perm_permission FOREIGN KEY (permission_id)
    REFERENCES sys_permission(id) ON UPDATE CASCADE ON DELETE CASCADE
);

ALTER TABLE sys_department
  ADD CONSTRAINT fk_sys_department_leader FOREIGN KEY (leader_user_id)
  REFERENCES sys_user(id) ON UPDATE CASCADE ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_sys_department_parent_id ON sys_department(parent_id);
CREATE INDEX IF NOT EXISTS idx_sys_department_leader_user_id ON sys_department(leader_user_id);
CREATE INDEX IF NOT EXISTS idx_sys_department_status ON sys_department(status);
CREATE INDEX IF NOT EXISTS idx_sys_department_is_deleted ON sys_department(is_deleted);

CREATE INDEX IF NOT EXISTS idx_sys_user_department_id ON sys_user(department_id);
CREATE INDEX IF NOT EXISTS idx_sys_user_email ON sys_user(email);
CREATE INDEX IF NOT EXISTS idx_sys_user_phone ON sys_user(phone);
CREATE INDEX IF NOT EXISTS idx_sys_user_status ON sys_user(status);
CREATE INDEX IF NOT EXISTS idx_sys_user_is_deleted ON sys_user(is_deleted);

CREATE INDEX IF NOT EXISTS idx_sys_role_status ON sys_role(status);
CREATE INDEX IF NOT EXISTS idx_sys_role_is_deleted ON sys_role(is_deleted);

CREATE INDEX IF NOT EXISTS idx_sys_permission_resource ON sys_permission(resource);
CREATE INDEX IF NOT EXISTS idx_sys_permission_action ON sys_permission(action);
CREATE INDEX IF NOT EXISTS idx_sys_permission_is_deleted ON sys_permission(is_deleted);

CREATE INDEX IF NOT EXISTS idx_sys_user_role_user_id ON sys_user_role(user_id);
CREATE INDEX IF NOT EXISTS idx_sys_user_role_role_id ON sys_user_role(role_id);
CREATE INDEX IF NOT EXISTS idx_sys_user_role_fund_scope ON sys_user_role USING GIN (fund_scope);
CREATE INDEX IF NOT EXISTS idx_sys_user_role_project_scope ON sys_user_role USING GIN (project_scope);
CREATE INDEX IF NOT EXISTS idx_sys_user_role_is_deleted ON sys_user_role(is_deleted);

CREATE INDEX IF NOT EXISTS idx_sys_role_perm_role_id ON sys_role_perm(role_id);
CREATE INDEX IF NOT EXISTS idx_sys_role_perm_permission_id ON sys_role_perm(permission_id);
CREATE INDEX IF NOT EXISTS idx_sys_role_perm_is_deleted ON sys_role_perm(is_deleted);

INSERT INTO sys_department (dept_code, dept_name, sort_order, status)
VALUES ('ROOT', 'System Administration', 0, 'active')
ON CONFLICT (dept_code) DO NOTHING;

INSERT INTO sys_role (role_code, role_name, description, status) VALUES
('SYS_ADMIN', 'System Admin', 'Full platform administrator', 'active'),
('ORG_ADMIN', 'Organization Admin', 'Manage departments and users', 'active'),
('COMPLIANCE_MANAGER', 'Compliance Manager', 'Review compliance sensitive operations', 'active'),
('RISK_MANAGER', 'Risk Manager', 'Risk review role', 'active'),
('FUND_MANAGER', 'Fund Manager', 'Fund module owner', 'active'),
('INVEST_MANAGER', 'Investment Manager', 'Investment module owner', 'active'),
('PROJECT_MANAGER', 'Project Manager', 'Project module owner', 'active'),
('MEETING_MANAGER', 'Meeting Manager', 'Meeting module owner', 'active'),
('EXIT_MANAGER', 'Exit Manager', 'Exit module owner', 'active'),
('AGENT_MANAGER', 'Agent Manager', 'AI agent module owner', 'active'),
('VIEWER', 'Viewer', 'Read-only base system user', 'active')
ON CONFLICT (role_code) DO NOTHING;

INSERT INTO sys_permission (perm_code, perm_name, resource, action, description) VALUES
('*', 'All permissions', '*', '*', 'All permissions'),
('system:user:manage', 'Manage users', 'system_user', 'manage', 'Manage users'),
('system:role:manage', 'Manage roles', 'system_role', 'manage', 'Manage roles'),
('audit:read', 'Read audit logs', 'audit', 'read', 'Read audit logs'),
('risk:read', 'Read risk data', 'risk', 'read', 'Read risk data'),
('fund:manage', 'Manage funds', 'fund', 'manage', 'Manage funds'),
('invest:manage', 'Manage investments', 'invest', 'manage', 'Manage investments'),
('project:manage', 'Manage projects', 'project', 'manage', 'Manage projects'),
('meeting:manage', 'Manage meetings', 'meeting', 'manage', 'Manage meetings'),
('exit:manage', 'Manage exits', 'exit', 'manage', 'Manage exits'),
('agent:manage', 'Manage agents', 'agent', 'manage', 'Manage agents'),
('base:read', 'Read base resources', 'base', 'read', 'Read base resources')
ON CONFLICT (perm_code) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = '*'
WHERE r.role_code = 'SYS_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code IN ('system:user:manage', 'system:role:manage')
WHERE r.role_code = 'ORG_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = 'audit:read'
WHERE r.role_code = 'COMPLIANCE_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = 'risk:read'
WHERE r.role_code = 'RISK_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = 'fund:manage'
WHERE r.role_code = 'FUND_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = 'invest:manage'
WHERE r.role_code = 'INVEST_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = 'project:manage'
WHERE r.role_code = 'PROJECT_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = 'meeting:manage'
WHERE r.role_code = 'MEETING_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = 'exit:manage'
WHERE r.role_code = 'EXIT_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = 'agent:manage'
WHERE r.role_code = 'AGENT_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = 'base:read'
WHERE r.role_code = 'VIEWER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_user (username, password, real_name, email, status, department_id)
SELECT
  'admin',
  '$2a$10$tOdipUFcODKLMZwKnoCMyucPPKi4I7TftT0DWc3YgV7jHsB6YPx3u',
  'System Administrator',
  'admin@example.com',
  'active',
  d.id
FROM sys_department d
WHERE d.dept_code = 'ROOT'
ON CONFLICT (username) DO NOTHING;

INSERT INTO sys_user_role (user_id, role_id)
SELECT u.id, r.id
FROM sys_user u
JOIN sys_role r ON r.role_code = 'SYS_ADMIN'
WHERE u.username = 'admin'
ON CONFLICT (user_id, role_id) DO NOTHING;

-- Align the bootstrap RBAC matrix with the architecture document's 11 roles.
UPDATE sys_role
SET status = 'inactive', is_deleted = TRUE
WHERE role_code IN ('ORG_ADMIN', 'COMPLIANCE_MANAGER', 'PROJECT_MANAGER', 'MEETING_MANAGER', 'EXIT_MANAGER', 'AGENT_MANAGER', 'VIEWER');

INSERT INTO sys_role (role_code, role_name, description, status) VALUES
('SYS_ADMIN', '系统管理员', '用户管理、角色权限配置、系统配置、审计日志', 'active'),
('FUND_MANAGER', '基金管理人', '基金设立、LP管理、募资管理、合规审查', 'active'),
('INVEST_MANAGER', '投资经理', '项目筛选、尽职调查、投决发起、投资执行', 'active'),
('POST_INVEST_MANAGER', '投后经理', '投后监控、风险预警、报告管理、退出管理', 'active'),
('RISK_MANAGER', '风控经理', '风控审核、风险评级、缓释措施', 'active'),
('LEGAL_MANAGER', '法务经理', '合同审查、协议起草、条款对比', 'active'),
('FINANCE_MANAGER', '财务经理', '财务报表、估值计算、收益分配', 'active'),
('OPS_MANAGER', '运营经理', '会议管理、知识库维护、内部协同', 'active'),
('REVIEWER', '复核人员', 'AI生成内容复核、投资建议复核、估值复核', 'active'),
('LP', 'LP（投资者）', '查看基金报告、接收信披通知、参与合伙人会议', 'active'),
('GUEST', '访客', '查看公开信息（需授权）', 'active')
ON CONFLICT (role_code) DO UPDATE
SET role_name = EXCLUDED.role_name,
    description = EXCLUDED.description,
    status = 'active',
    is_deleted = FALSE;

INSERT INTO sys_permission (perm_code, perm_name, resource, action, description) VALUES
('lp:manage', 'LP管理', 'lp', 'manage', 'LP管理'),
('meeting:read', '会议查看', 'meeting', 'read', '会议查看'),
('invest:read', '投资数据查看', 'invest', 'read', '投资数据查看'),
('risk:manage', '风控管理', 'risk', 'manage', '风控管理'),
('legal:manage', '法务管理', 'legal', 'manage', '法务管理'),
('contract:manage', '合同管理', 'contract', 'manage', '合同管理'),
('finance:manage', '财务管理', 'finance', 'manage', '财务管理'),
('valuation:manage', '估值管理', 'valuation', 'manage', '估值管理'),
('knowledge:manage', '知识库管理', 'knowledge', 'manage', '知识库管理'),
('review:manage', '复核管理', 'review', 'manage', '复核管理'),
('lp_portal:read', 'LP门户查看', 'lp_portal', 'read', 'LP门户查看'),
('public:read', '公开信息查看', 'public', 'read', '公开信息查看')
ON CONFLICT (perm_code) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code IN ('fund:manage', 'lp:manage', 'audit:read')
WHERE r.role_code = 'FUND_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code IN ('invest:manage', 'meeting:read', 'audit:read')
WHERE r.role_code = 'INVEST_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code IN ('project:manage', 'exit:manage', 'audit:read')
WHERE r.role_code = 'POST_INVEST_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code IN ('risk:manage', 'invest:read', 'audit:read')
WHERE r.role_code = 'RISK_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code IN ('legal:manage', 'contract:manage', 'audit:read')
WHERE r.role_code = 'LEGAL_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code IN ('finance:manage', 'valuation:manage', 'audit:read')
WHERE r.role_code = 'FINANCE_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code IN ('meeting:manage', 'knowledge:manage', 'audit:read')
WHERE r.role_code = 'OPS_MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code IN ('review:manage', 'audit:read')
WHERE r.role_code = 'REVIEWER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code IN ('lp_portal:read', 'audit:read')
WHERE r.role_code = 'LP'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO sys_role_perm (role_id, permission_id)
SELECT r.id, p.id FROM sys_role r JOIN sys_permission p ON p.perm_code = 'public:read'
WHERE r.role_code = 'GUEST'
ON CONFLICT (role_id, permission_id) DO NOTHING;
