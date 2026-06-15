package config

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	_ "github.com/go-sql-driver/mysql"

	"github.com/tifacusil/go-project-management-service/migrations"
)

// MigrateUp applies all pending goose migrations as root (DDL), then closes the
// root connection. goose es la fuente de verdad del schema (CLAUDE.md §2.4): el
// infra solo crea DB + usuario; este servicio crea/actualiza las tablas.
//
// Idempotente: las migraciones usan CREATE TABLE IF NOT EXISTS, así que correr
// sobre una DB con tablas pre-existentes no las clobberea ni pierde datos.
func MigrateUp(c Config) error {
	db, err := sql.Open("mysql", c.RootDSN())
	if err != nil {
		return fmt.Errorf("open root conn: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
