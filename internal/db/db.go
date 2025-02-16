package db

import (
	"os"

	"github.com/aybolid/wishbot/internal/logger"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const DB_DIR = "data"
const DB_FILE = "wishbot.db"

var Database *sqlx.DB

// Initializes the database connection.
func Init() {
	var err error

	if _, err := os.Stat(DB_DIR); os.IsNotExist(err) {
		os.Mkdir(DB_DIR, 0755)
	}

	dbPath := DB_DIR + "/" + DB_FILE

	Database, err = sqlx.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}

	if err = Database.Ping(); err != nil {
		panic(err)
	}

	logger.Sugared.Infow("connected to database", "path", dbPath)

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
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY(owner_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- Group members table with a foreign key relation to groups.
CREATE TABLE IF NOT EXISTS group_members (
	member_id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY(group_id) REFERENCES groups(group_id) ON DELETE CASCADE,
	FOREIGN KEY(user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS group_members_unique_idx ON group_members (group_id, user_id);

-- Wishes table.
CREATE TABLE IF NOT EXISTS wishes (
    wish_id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
	member_id INTEGER NOT NULL,
	url TEXT NOT NULL,
	description TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY(group_id) REFERENCES groups(group_id) ON DELETE CASCADE,
    FOREIGN KEY(user_id) REFERENCES users(user_id) ON DELETE CASCADE,
	FOREIGN KEY(member_id) REFERENCES group_members(member_id) ON DELETE CASCADE
);
`

func runStartupMigrations() {
	Database.MustExec(schema)
	logger.Sugared.Infow("ran startup migrations")
}
