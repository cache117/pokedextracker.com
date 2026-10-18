package main

import (
	"github.com/go-pg/pg/v10/orm"
	migrations "github.com/robinjoseph08/go-pg-migrations/v3"
)

func init() {
	up := func(db orm.DB) error {
		_, err := db.Exec(`
 CREATE TABLE personal_catalog_settings (id INTEGER PRIMARY KEY CHECK(id=1), revision INTEGER NOT NULL DEFAULT 0, document JSONB NOT NULL);
 INSERT INTO personal_catalog_settings(id,document) VALUES(1,'{}');
 CREATE TABLE personal_location_overrides (
 pokemon_id INTEGER NOT NULL REFERENCES pokemon(id), game_id TEXT NOT NULL REFERENCES games(id),
 values TEXT[] NOT NULL, updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 PRIMARY KEY(pokemon_id,game_id));`)
		return err
	}
	down := func(db orm.DB) error {
		_, err := db.Exec(`DROP TABLE personal_location_overrides; DROP TABLE personal_catalog_settings;`)
		return err
	}
	migrations.Register("20261014000100_create_personal_catalog", up, down, migrations.MigrationOptions{})
}
