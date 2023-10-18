package db

import (
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4"
)

type Nest struct {
	Id      int         `db:"nest_id"`
	Lat     float64     `db:"lat"`
	Lon     float64     `db:"lon"`
	Name    null.String `db:"name"`
	Polygon string      `db:"polygon_astext"`
}

func LoadNests(db DbDetails) ([]Nest, error) {
	fortIds := []Nest{}
	err := db.GeneralDb.Select(&fortIds,
		"SELECT nest_id, lat, lon, name, st_astext(polygon) as polygon_astext FROM nests WHERE active = 1")
	if err != nil {
		return nil, err
	}

	return fortIds, nil
}

func SaveNest(db *sqlx.DB, area string, maxPokemonId int, maxPokemonCount int, pokemonAvg float64, ratio float64) error {
	nestData := map[string]interface{}{
		"pokemonId":    maxPokemonId,
		"pokemonCount": maxPokemonCount,
		"ratio":        ratio,
		"pokemonAvg":   pokemonAvg,
		"nestId":       area,
	}
	_, err := db.NamedExec("UPDATE nests SET pokemon_id = :pokemonId, pokemon_form = 0, pokemon_count = :pokemonCount, pokemon_avg = :pokemonAvg, pokemon_ratio = :ratio, updated = unix_timestamp() WHERE nest_id = :nestId and active = 1",
		nestData)
	if err != nil {
		return err
	}

	return nil
}
