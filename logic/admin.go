package logic

import (
	"fmt"

	"github.com/zorchenhimer/MoviePolls/models"
)

// Purge removes the account entirely, including all of the account's votes.
// Should this add the user to the banlist?  Maybe add an option?
func (b *backend) AdminPurgeUser(user *models.User) error {
	b.l.Info("Purging user %s", user)
	err := b.data.PurgeUser(user.Id)
	if err != nil {
		return err
	}
	return nil
}

// Ban deletes a user and adds them to a ban list.  Users on this list can view
// the site but cannot create an account.
func (b *backend) AdminBanUser(user *models.User) error {
	return fmt.Errorf("not implemented")
}

func (s *backend) CheckAdminRights(user *models.User) bool {
	ok := true
	if user == nil || user.Privilege < models.PRIV_MOD {
		ok = false
	}
	return ok
}

// "deletes" a user.  The account will still exist along with the votes, but
// the name, password, email, and notification settings will all be removed.
func (s *backend) AdminDeleteUser(user *models.User) error {
	s.l.Info("Deleting user %s", user)
	user.Name = "[deleted]"
	for _, auth := range user.AuthMethods {
		s.data.DeleteAuthMethod(auth.Id)
	}
	user.AuthMethods = []*models.AuthMethod{}
	user.Email = ""
	user.NotifyCycleEnd = false
	user.NotifyVoteSelection = false
	user.Privilege = 0

	return s.data.UpdateUser(user)

}
