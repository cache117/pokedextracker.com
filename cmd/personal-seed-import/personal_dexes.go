package main

import (
	_ "embed"

	"github.com/go-pg/pg/v10"
)

//go:embed fire_red_leaf_green.sql
var fireRedLeafGreenSQL string

func importPersonalDexes(tx *pg.Tx) error {
	_, err := tx.Exec(fireRedLeafGreenSQL)
	if err != nil {
		return err
	}
	return importPersonalLocations(tx)
}
