package db

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
	ID          int64
	GroupID     int64
	UserID      int64
	URL         string
	Description string
	CreatedAt   string
	UpdatedAt   string
}

func CreateWish(url string, desc string, userId int64, groupId int64) (*Wish, error) {
	tx, err := Database.Beginx()
	if err != nil {
		return nil, err
	}

	insertQuery := "INSERT INTO wishes (url, description, user_id, group_id) VALUES (?, ?, ?, ?)"
	result, err := tx.Exec(insertQuery, url, desc, userId, groupId)
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

	return dbw.ToWish(), nil
}

func (dbw *dbWish) ToWish() *Wish {
	return &Wish{
		ID:          dbw.WishID,
		GroupID:     dbw.GroupID,
		UserID:      dbw.UserID,
		URL:         dbw.URL,
		Description: dbw.Description,
		CreatedAt:   dbw.CreatedAt,
		UpdatedAt:   dbw.UpdatedAt,
	}
}
