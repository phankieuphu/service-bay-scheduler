package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Appointment struct {
	ID             int64           `gorm:"column:id;type:bigint;primaryKey"`
	DealershipID   int64           `gorm:"column:dealership_id;type:bigint"`
	CustomerID     int64           `gorm:"column:customer_id;type:bigint"`
	ScheduledStart time.Time       `gorm:"column:scheduled_start;type:timestamptz"`
	ScheduledEnd   time.Time       `gorm:"column:scheduled_end;type:timestamptz"`
	Status         string          `gorm:"column:status;type:varchar;size:20"`
	TotalAmount    decimal.Decimal `gorm:"column:total_amount;type:numeric(12,2)"`
	CreatedAt      time.Time       `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt      time.Time       `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`
}

func (Appointment) TableName() string { return "appointment" }

// AppointmentBay leaves out the generated slot column, which can't be
// written.
type AppointmentBay struct {
	ID            int64     `gorm:"column:id;type:bigint;primaryKey"`
	AppointmentID int64     `gorm:"column:appointment_id;type:bigint"`
	ServiceBayID  int64     `gorm:"column:service_bay_id;type:bigint"`
	StartTime     time.Time `gorm:"column:start_time;type:timestamptz"`
	EndTime       time.Time `gorm:"column:end_time;type:timestamptz"`
	Status        string    `gorm:"column:status;type:varchar;size:20"`
}

func (AppointmentBay) TableName() string { return "appointment_bay" }

type AppointmentVehicle struct {
	ID                int64  `gorm:"column:id;type:bigint;primaryKey"`
	AppointmentID     int64  `gorm:"column:appointment_id;type:bigint"`
	CustomerVehicleID int64  `gorm:"column:customer_vehicle_id;type:bigint"`
	Status            string `gorm:"column:status;type:varchar;size:20"`
}

func (AppointmentVehicle) TableName() string { return "appointment_vehicle" }

type AppointmentService struct {
	ID                   int64           `gorm:"column:id;type:bigint;primaryKey"`
	AppointmentID        int64           `gorm:"column:appointment_id;type:bigint"`
	ServiceID            int64           `gorm:"column:service_id;type:bigint"`
	AppointmentVehicleID *int64          `gorm:"column:appointment_vehicle_id;type:bigint"`
	AppointmentBayID     *int64          `gorm:"column:appointment_bay_id;type:bigint"`
	FeeCharged           decimal.Decimal `gorm:"column:fee_charged;type:numeric(12,2)"`
	Status               string          `gorm:"column:status;type:varchar;size:20"`
}

func (AppointmentService) TableName() string { return "appointment_service" }
