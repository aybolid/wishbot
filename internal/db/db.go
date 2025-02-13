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

	runStartupMigrations()
}

var schema = `
-- Enable foreign key constraints.
PRAGMA foreign_keys = ON;

-- Users table.
CREATE TABLE IF NOT EXISTS users (
	user_id INTEGER PRIMARY KEY, -- telegram user id
	username TEXT NOT NULL UNIQUE,
	chat_id INTEGER NOT NULL UNIQUE,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Groups table.
CREATE TABLE IF NOT EXISTS groups (
    group_id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Group members table with a foreign key relation to groups.
CREATE TABLE IF NOT EXISTS group_members (
    group_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY(group_id) REFERENCES groups(group_id) ON DELETE CASCADE
);
`

func runStartupMigrations() {
	DB.MustExec(schema)
	logger.SUGAR.Infow("ran startup migrations")
}
