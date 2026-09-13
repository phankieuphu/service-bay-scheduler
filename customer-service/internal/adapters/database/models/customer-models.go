package models

import (
	"customer-service/internal/constants"
	"time"
)

type Customer struct {
	ID        int64                    `json:"id" gorm:"column:id;type:bigint"`
	Name      string                   `json:"name" gorm:"column:name;type:varchar;size:255"`
	Email     string                   `json:"email" gorm:"column:email;type:varchar;size:255"`
	Phone     string                   `json:"phone" gorm:"column:phone;type:varchar;size:30"`
	BirthDay  time.Time                `json:"birth_day" gorm:"column:birth_day;type:date"`
	Status    constants.CustomerStatus `json:"status" gorm:"column:status;type:varchar;size:20"`
	CreatedAt time.Time                `json:"created_at" gorm:"column:created_at;type:timestamp;autoCreateTime"`
	UpdatedAt time.Time                `json:"updated_at" gorm:"column:updated_at;type:timestamp;autoUpdateTime"`
}

func (c Customer) TableName() string {
	return "customer"
}
