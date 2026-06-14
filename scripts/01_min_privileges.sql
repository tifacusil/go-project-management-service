-- Least-privilege DB user for project-management-service (CLAUDE.md §1.2, §2.4).
--
-- Run ONCE as a DB admin. The service connects as `proj-manager` and is granted
-- ONLY the operations it actually performs on ONLY its own 3 tables:
--   SELECT, INSERT, UPDATE  -- soft delete is an UPDATE, so DELETE is NOT granted.
--
-- TODO(SE-practice): database-per-service. Baseline shares one MySQL across the
-- 6 services (Shared Persistency bad smell). Strangler Fig cutover to a dedicated
-- MySQL instance per service is pending; until then this scopes blast radius.

CREATE USER IF NOT EXISTS 'proj-manager'@'%' IDENTIFIED BY 'CHANGE_ME';

GRANT SELECT, INSERT, UPDATE ON project_management_db.projects   TO 'proj-manager'@'%';
GRANT SELECT, INSERT, UPDATE ON project_management_db.sensors    TO 'proj-manager'@'%';
GRANT SELECT, INSERT, UPDATE ON project_management_db.equipment  TO 'proj-manager'@'%';

-- goose needs its bookkeeping table; grant DDL+DML only on it.
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE ON project_management_db.goose_db_version TO 'proj-manager'@'%';

FLUSH PRIVILEGES;
