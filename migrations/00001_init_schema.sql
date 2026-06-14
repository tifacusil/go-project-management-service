-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS projects (
    project_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    location_lat DOUBLE,
    location_lng DOUBLE,
    status ENUM('active','inactive') NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS sensors (
    sensor_id VARCHAR(64) PRIMARY KEY,
    project_id VARCHAR(64) NOT NULL,
    sensor_type VARCHAR(32) NOT NULL,
    model VARCHAR(128),
    install_lat DOUBLE,
    install_lng DOUBLE,
    install_depth DOUBLE,
    status ENUM('active','inactive') NOT NULL DEFAULT 'active',
    installed_at DATETIME,
    INDEX idx_sensors_project (project_id),
    INDEX idx_sensors_project_status (project_id, status)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS equipment (
    equipment_id VARCHAR(64) PRIMARY KEY,
    project_id VARCHAR(64) NOT NULL,
    type VARCHAR(64),
    model VARCHAR(128),
    serial VARCHAR(128),
    status ENUM('active','inactive') NOT NULL DEFAULT 'active',
    INDEX idx_equipment_project (project_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS equipment;
DROP TABLE IF EXISTS sensors;
DROP TABLE IF EXISTS projects;
-- +goose StatementEnd
