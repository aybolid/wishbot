package tgbot

import "github.com/aybolid/wishbot/internal/logger"

type botState struct {
	// pendingGroupCreation tracks users that are currently creating a group.
	pendingGroupCreation map[int64]bool
}

var state = &botState{
	pendingGroupCreation: make(map[int64]bool),
}

// isPendingGroupCreation returns true if a user is currently creating a group.
func (s *botState) isPendingGroupCreation(userID int64) bool {
	_, ok := s.pendingGroupCreation[userID]
	logger.SUGAR.Infow("is pending group creation", "user_id", userID, "pending", ok)
	return ok
}

// setPendingGroupCreation marks a user as pending group creation. Releases the user beforehand.
func (s *botState) setPendingGroupCreation(userID int64) {
	s.releaseUser(userID)
	logger.SUGAR.Infow("setting pending group creation", "user_id", userID)
	s.pendingGroupCreation[userID] = true
}

// releaseUser releases a user from pending flows.
func (s *botState) releaseUser(userID int64) {
	logger.SUGAR.Infow("releasing user", "user_id", userID)
	delete(s.pendingGroupCreation, userID)
}
