package users

import "time"

type User struct {
	ID        int64
	Name      string
	Email     string
	Role      string
	CreatedAt time.Time
}
