package db

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type dbUser struct {
	UserID int64 `db:"user_id"`
	// Username is stored without the @ symbol.
	Username  string `db:"username"`
	ChatID    int64  `db:"chat_id"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

type User struct {
	UserID int64
	// Username is stored without the @ symbol.
	Username  string
	ChatID    int64
	CreatedAt string
	UpdatedAt string
}

// GetUser returns a user by user id.
func GetUser(userID int64) (*User, error) {
	var dbUser dbUser

	query := "SELECT * FROM users WHERE user_id = ?"
	if err := Database.Get(&dbUser, query, userID); err != nil {
		return nil, err
	}

	return dbUser.toUser(), nil
}

// GetUserByUsername returns a user by username.
// Username is stored without the @ symbol.
func GetUserByUsername(username string) (*User, error) {
	var dbUser dbUser

	query := "SELECT * FROM users WHERE username = ?"
	if err := Database.Get(&dbUser, query, username); err != nil {
		return nil, err
	}

	return dbUser.toUser(), nil
}

// CreateUser creates a new user in the database.
func CreateUser(user *tgbotapi.User, chatID int64) (*User, error) {
	tx, err := Database.Beginx()
	if err != nil {
		return nil, err
	}

	insertQuery := "INSERT INTO users (user_id, username, chat_id) VALUES (?, ?, ?)"
	if _, err := tx.Exec(insertQuery, user.ID, user.UserName, chatID); err != nil {
		tx.Rollback()
		return nil, err
	}

	dbu := &dbUser{}
	selectQuery := "SELECT * FROM users WHERE user_id = ?"
	if err := tx.Get(dbu, selectQuery, user.ID); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return dbu.toUser(), nil
}

func (dbu *dbUser) toUser() *User {
	return &User{
		UserID:    dbu.UserID,
		Username:  dbu.Username,
		ChatID:    dbu.ChatID,
		CreatedAt: dbu.CreatedAt,
		UpdatedAt: dbu.UpdatedAt,
	}
}
