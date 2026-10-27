ALTER TABLE IF EXISTS sys_department DROP CONSTRAINT IF EXISTS fk_sys_department_leader;

DROP TABLE IF EXISTS sys_role_perm;
DROP TABLE IF EXISTS sys_user_role;
DROP TABLE IF EXISTS sys_permission;
DROP TABLE IF EXISTS sys_role;
DROP TABLE IF EXISTS sys_user;
DROP TABLE IF EXISTS sys_department;
