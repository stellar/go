package scenarios

import (
	"log"

	"github.com/jmoiron/sqlx"
)

//go:generate go-bindata -ignore (go|rb)$ -pkg scenarios .

// Load executes the sql script at `path` on postgres database at `url`
func Load(db *sqlx.DB, url string, path string) {
	sql, err := Asset(path)
	if err != nil {
		log.Panic(err)
	}

	_, err = db.Exec(string(sql))
	if err != nil {
		log.Panic(err)
	}

}
