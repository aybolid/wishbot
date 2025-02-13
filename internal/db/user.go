package db

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type dbUser struct {
	UserID    int64  `db:"user_id"`
	Username  string `db:"username"`
	ChatID    int64  `db:"chat_id"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

type User struct {
	UserID    int64
	Username  string
	ChatID    int64
	CreatedAt string
	UpdatedAt string
}

func GetUser(userID int64) (*User, error) {
	var dbUser dbUser

	query := "SELECT * FROM users WHERE user_id = ?"
	if err := DB.Get(&dbUser, query, userID); err != nil {
		return nil, err
	}

	return dbUser.ToUser(), nil
}

func GetUserByUsername(username string) (*User, error) {
	var dbUser dbUser

	query := "SELECT * FROM users WHERE username = ?"
	if err := DB.Get(&dbUser, query, username); err != nil {
		return nil, err
	}

	return dbUser.ToUser(), nil
}

func CreateUser(user *tgbotapi.User, chatID int64) (*User, error) {
	tx, err := DB.Beginx()
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

	return dbu.ToUser(), nil
}

func (dbu *dbUser) ToUser() *User {
	return &User{
		UserID:    dbu.UserID,
		Username:  dbu.Username,
		ChatID:    dbu.ChatID,
		CreatedAt: dbu.CreatedAt,
		UpdatedAt: dbu.UpdatedAt,
	}
}
