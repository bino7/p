package p

import (
	_ "github.com/cayleygraph/cayley/graph/bolt"
	"github.com/asdine/storm"
)

var (
	store_path = "p.db"
	db *storm.DB
)

func initStore() {
	d, err := storm.Open(store_path)
	if err != nil {
		panic(err)
	}
	db = d

	if err := db.Init(&User{}); err != nil {
		panic(err)
	}
}