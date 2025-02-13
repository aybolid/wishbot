package tgbot

import "github.com/aybolid/wishbot/internal/logger"

type botState struct {
	// pendingGroupCreation tracks users that are currently creating a group.
	// user id -> bool
	pendingGroupCreation map[int64]bool
	// pendingInviteCreation tracks users that are currently creating an invite.
	// user id -> bool
	pendingInviteCreation map[int64]bool
}

var state = &botState{
	pendingGroupCreation:  make(map[int64]bool),
	pendingInviteCreation: make(map[int64]bool),
}

// isPendingGroupCreation returns true if a user is currently creating a group.
func (s *botState) isPendingGroupCreation(userID int64) bool {
	_, ok := s.pendingGroupCreation[userID]
	logger.SUGAR.Infow("is pending group creation", "user_id", userID, "pending", ok)
	return ok
}

// isPendingInviteCreation returns true if a user is currently creating an invite.
func (s *botState) isPendingInviteCreation(userID int64) bool {
	_, ok := s.pendingInviteCreation[userID]
	logger.SUGAR.Infow("is pending invite creation", "user_id", userID, "pending", ok)
	return ok
}

// setPendingGroupCreation marks a user as pending group creation. Releases the user beforehand.
func (s *botState) setPendingGroupCreation(userID int64) {
	s.releaseUser(userID)
	logger.SUGAR.Infow("setting pending group creation", "user_id", userID)
	s.pendingGroupCreation[userID] = true
}

// setPendingInviteCreation marks a user as pending invite creation.
// Releases the user beforehand.
func (s *botState) setPendingInviteCreation(userID int64) {
	s.releaseUser(userID)
	logger.SUGAR.Infow("setting pending invite creation", "user_id", userID)
	s.pendingInviteCreation[userID] = true
}

// releaseUser releases a user from pending flows.
func (s *botState) releaseUser(userID int64) {
	logger.SUGAR.Infow("releasing user", "user_id", userID)
	delete(s.pendingGroupCreation, userID)
}
