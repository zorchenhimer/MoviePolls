package common

import (
	"fmt"
)

type ErrNoUsersFound struct {
	Auth AuthType
}

func (e *ErrNoUsersFound) Error() string {
	return fmt.Sprintf("No users found with AuthType %v", e.Auth)
}
