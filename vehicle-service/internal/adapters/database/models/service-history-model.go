package models

import "time"

type ServiceHistory struct {
	ID            int64     `json:"id" gorm:"column:id;type:bigint;<-:false"`
	VehicleID     int64     `json:"vehicle_id" gorm:"column:vehicle_id;type:bigint"`
	AppointmentID int64     `json:"appointment_id" gorm:"column:appointment_id;type:bigint"`
	DealershipID  int64     `json:"dealership_id" gorm:"column:dealership_id;type:bigint"`
	CompletedAt   time.Time `json:"completed_at" gorm:"column:completed_at;type:timestamptz"`
	Services      []byte    `json:"services" gorm:"column:services;type:jsonb"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at;type:timestamptz;autoCreateTime"`
}

func (ServiceHistory) TableName() string {
	return "vehicle_service_history"
}
