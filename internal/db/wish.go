package db

import "github.com/aybolid/wishbot/internal/logger"

type dbWish struct {
	WishID      int64  `db:"wish_id"`
	GroupID     int64  `db:"group_id"`
	UserID      int64  `db:"user_id"`
	URL         string `db:"url"`
	Description string `db:"description"`
	CreatedAt   string `db:"created_at"`
	UpdatedAt   string `db:"updated_at"`
}

type Wish struct {
	WishID      int64
	GroupID     int64
	UserID      int64
	URL         string
	Description string
	CreatedAt   string
	UpdatedAt   string
}

// GetGroupWishes retrieves all wishes for a given group.
func GetGroupWishes(groupID int64) ([]*Wish, error) {
	logger.Sugared.Infow("getting group wishes", "group_id", groupID)

	var dbWishes []*dbWish

	selectQuery := "SELECT * FROM wishes WHERE group_id = ?"
	err := Database.Select(&dbWishes, selectQuery, groupID)
	if err != nil {
		return nil, err
	}

	wishes := make([]*Wish, len(dbWishes))
	for idx, dbw := range dbWishes {
		wishes[idx] = dbw.toWish()
	}

	return wishes, nil
}

// CreateWish creates a new wish for a given user and group.
func CreateWish(url string, desc string, userID int64, groupID int64) (*Wish, error) {
	logger.Sugared.Infow("creating wish", "url", url, "description", desc, "user_id", userID, "group_id", groupID)

	tx, err := Database.Beginx()
	if err != nil {
		return nil, err
	}

	insertQuery := "INSERT INTO wishes (url, description, user_id, group_id) VALUES (?, ?, ?, ?)"
	result, err := tx.Exec(insertQuery, url, desc, userID, groupID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	wishID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	dbw := &dbWish{}
	selectQuery := "SELECT * FROM wishes WHERE wish_id = ?"
	if err := tx.Get(dbw, selectQuery, wishID); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return dbw.toWish(), nil
}

func (dbw *dbWish) toWish() *Wish {
	return &Wish{
		WishID:      dbw.WishID,
		GroupID:     dbw.GroupID,
		UserID:      dbw.UserID,
		URL:         dbw.URL,
		Description: dbw.Description,
		CreatedAt:   dbw.CreatedAt,
		UpdatedAt:   dbw.UpdatedAt,
	}
}
