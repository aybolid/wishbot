package db

import (
	"github.com/aybolid/wishbot/internal/env"
	"github.com/aybolid/wishbot/internal/logger"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sqlx.DB

// Initializes the database connection.
func Init() {
	var err error

	DB, err = sqlx.Open("sqlite3", env.VARS.DBPath)
	if err != nil {
		panic(err)
	}

	if err = DB.Ping(); err != nil {
		panic(err)
	}

	logger.SUGAR.Infow("connected to database", "path", env.VARS.DBPath)
}
