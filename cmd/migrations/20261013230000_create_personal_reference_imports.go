package main

import (
	"github.com/go-pg/pg/v10/orm"
	migrations "github.com/robinjoseph08/go-pg-migrations/v3"
)

func init() {
	up := func(db orm.DB) error {
		_, err := db.Exec(`CREATE TABLE personal_reference_imports (
  pokemon_id INTEGER PRIMARY KEY REFERENCES pokemon(id),
  source_url TEXT NOT NULL,
  retrieved_at TIMESTAMPTZ NOT NULL,
  sha256 TEXT NOT NULL
 )`)
		return err
	}
	down := func(db orm.DB) error { _, err := db.Exec(`DROP TABLE personal_reference_imports`); return err }
	migrations.Register("20261013230000_create_personal_reference_imports", up, down, migrations.MigrationOptions{})
}
