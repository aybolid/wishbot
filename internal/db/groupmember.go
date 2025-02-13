package db

import "github.com/aybolid/wishbot/internal/logger"

type dbGroupMember struct {
	GroupID   int64  `db:"group_id"`
	UserID    int64  `db:"user_id"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

type GroupMember struct {
	GroupID   int64
	UserID    int64
	CreatedAt string
	UpdatedAt string
}

// GetGroupMembers retrieves all members of a given group.
func GetGroupMembers(groupID int64) ([]*GroupMember, error) {
	logger.SUGAR.Infow("getting group members", "group_id", groupID)

	var dbMembers []dbGroupMember
	query := "SELECT * FROM group_members WHERE group_id = ?"
	if err := DB.Select(&dbMembers, query, groupID); err != nil {
		return nil, err
	}

	members := make([]*GroupMember, len(dbMembers))
	for i, dbm := range dbMembers {
		members[i] = dbm.ToGroupMember()
	}

	return members, nil
}

// CreateGroupMember inserts a new member into a group and returns the inserted row.
// The operation is wrapped in a transaction to allow rollback in case of an error.
func CreateGroupMember(groupID int64, userID int64) (*GroupMember, error) {
	logger.SUGAR.Infow("creating group member", "group_id", groupID, "user_id", userID)

	tx, err := DB.Beginx()
	if err != nil {
		return nil, err
	}

	insertQuery := "INSERT INTO group_members (group_id, user_id) VALUES (?, ?)"
	if _, err := tx.Exec(insertQuery, groupID, userID); err != nil {
		tx.Rollback()
		return nil, err
	}

	dbm := &dbGroupMember{}
	selectQuery := "SELECT * FROM group_members WHERE group_id = ? AND user_id = ?"
	if err := tx.Get(dbm, selectQuery, groupID, userID); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return dbm.ToGroupMember(), nil
}

func (dbm *dbGroupMember) ToGroupMember() *GroupMember {
	return &GroupMember{
		GroupID:   dbm.GroupID,
		UserID:    dbm.UserID,
		CreatedAt: dbm.CreatedAt,
		UpdatedAt: dbm.UpdatedAt,
	}
}
