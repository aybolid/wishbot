package db

import "github.com/aybolid/wishbot/internal/logger"

type dbGroup struct {
	ID        int64  `db:"group_id"`
	Name      string `db:"name"`
	OwnerID   int64  `db:"owner_id"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

type Group struct {
	ID        int64
	Name      string
	OwnerID   int64
	CreatedAt string
	UpdatedAt string
}

// GetUserGroups retrieves groups for a user by joining groups with group_members.
func GetUserGroups(userID int64) ([]*Group, error) {
	logger.SUGAR.Infow("getting user groups", "user_id", userID)

	var dbGroups []dbGroup
	query := `
        SELECT g.*
        FROM groups g
        INNER JOIN group_members gm ON g.group_id = gm.group_id
        WHERE gm.user_id = ?
    `
	if err := DB.Select(&dbGroups, query, userID); err != nil {
		return nil, err
	}

	groups := make([]*Group, len(dbGroups))
	for i, dbg := range dbGroups {
		groups[i] = dbg.ToGroup()
	}

	return groups, nil
}

// CreateGroup creates a new group and automatically adds the owner as a member.
// The entire operation is wrapped in a transaction to ensure atomicity.
func CreateGroup(ownerID int64, name string) (*Group, error) {
	logger.SUGAR.Infow("creating group", "owner_id", ownerID, "name", name)

	tx, err := DB.Beginx()
	if err != nil {
		return nil, err
	}

	groupInsertQuery := "INSERT INTO groups (owner_id, name) VALUES (?, ?)"
	result, err := tx.Exec(groupInsertQuery, ownerID, name)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	groupID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	memberInsertQuery := "INSERT INTO group_members (group_id, user_id) VALUES (?, ?)"
	if _, err := tx.Exec(memberInsertQuery, groupID, ownerID); err != nil {
		tx.Rollback()
		return nil, err
	}

	dbg := &dbGroup{}
	selectQuery := "SELECT * FROM groups WHERE group_id = ?"
	if err := tx.Get(dbg, selectQuery, groupID); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return dbg.ToGroup(), nil
}

func (dbg *dbGroup) ToGroup() *Group {
	return &Group{
		ID:        dbg.ID,
		Name:      dbg.Name,
		OwnerID:   dbg.OwnerID,
		CreatedAt: dbg.CreatedAt,
		UpdatedAt: dbg.UpdatedAt,
	}
}
