package tgbot

import "github.com/aybolid/wishbot/internal/logger"

type botState struct {
	// PendingGroupCreation tracks users that are currently creating a group.
	// user id -> bool
	PendingGroupCreation map[int64]bool
	// PendingInviteCreation tracks users that are currently creating an invite.
	// user id -> group id
	PendingInviteCreation map[int64]int64
}

// Inner state of the bot.
// User can only be in one of the actions at a time.
var State = &botState{
	PendingGroupCreation:  make(map[int64]bool),
	PendingInviteCreation: make(map[int64]int64),
}

// isPendingGroupCreation returns true if a user is currently creating a group.
func (s *botState) isPendingGroupCreation(userID int64) bool {
	_, ok := s.PendingGroupCreation[userID]
	logger.Sugared.Infow("is pending group creation", "user_id", userID, "pending", ok)
	return ok
}

// isPendingInviteCreation returns true if a user is currently creating an invite.
func (s *botState) isPendingInviteCreation(userID int64) bool {
	_, ok := s.PendingInviteCreation[userID]
	logger.Sugared.Infow("is pending invite creation", "user_id", userID, "pending", ok)
	return ok
}

// setPendingGroupCreation marks a user as pending group creation. Releases the user beforehand.
func (s *botState) setPendingGroupCreation(userID int64) {
	s.releaseUser(userID)
	logger.Sugared.Infow("setting pending group creation", "user_id", userID)
	s.PendingGroupCreation[userID] = true
}

// setPendingInviteCreation marks a user as pending invite creation.
// Releases the user beforehand.
func (s *botState) setPendingInviteCreation(userID int64, groupID int64) {
	s.releaseUser(userID)
	logger.Sugared.Infow("setting pending invite creation", "user_id", userID)
	s.PendingInviteCreation[userID] = groupID
}

// getPendingInviteCreation returns the group id for a user that is pending invite creation.
func getPendingInviteCreation(userID int64) (int64, bool) {
	groupID, ok := State.PendingInviteCreation[userID]
	return groupID, ok
}

// releaseUser releases a user from pending flows.
func (s *botState) releaseUser(userID int64) {
	logger.Sugared.Infow("releasing user", "user_id", userID)
	delete(s.PendingGroupCreation, userID)
	delete(s.PendingInviteCreation, userID)
}
