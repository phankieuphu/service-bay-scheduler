package models

import "time"

type User struct {
	ID           string    `json:"id" gorm:"column:id;type:uuid;primaryKey"`
	Email        string    `json:"email" gorm:"column:email;type:varchar;size:255"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;type:varchar;size:255"`
	Role         string    `json:"role" gorm:"column:role;type:varchar;size:20"`
	Status       string    `json:"status" gorm:"column:status;type:varchar;size:20"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;type:timestamptz"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at;type:timestamptz"`
}

// TableName is app_user because "user" is a reserved word in Postgres.
func (User) TableName() string {
	return "app_user"
}
