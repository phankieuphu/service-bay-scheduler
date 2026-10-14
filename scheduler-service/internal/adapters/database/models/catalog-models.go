package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Hub struct {
	ID        int64     `gorm:"column:id;type:bigint;primaryKey"`
	Name      string    `gorm:"column:name;type:varchar;size:255"`
	Address   string    `gorm:"column:address;type:varchar;size:255"`
	OpenHour  string    `gorm:"column:open_hour;type:time"`
	CloseHour string    `gorm:"column:close_hour;type:time"`
	TimeZone  string    `gorm:"column:timezone;type:varchar;size:64"`
	Active    bool      `gorm:"column:active;type:boolean"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz"`
	// BayCount isn't a column; ListHubs selects it.
	BayCount int `gorm:"column:bay_count;->"`
}

func (Hub) TableName() string { return "hub" }

type HubBay struct {
	ID        int64     `gorm:"column:id;type:bigint;primaryKey"`
	HubID     int64     `gorm:"column:hub_id;type:bigint"`
	Code      string    `gorm:"column:code;type:varchar;size:50"`
	Status    string    `gorm:"column:status;type:varchar;size:20"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz"`
}

func (HubBay) TableName() string { return "hub_bay" }

type CatalogService struct {
	ID              int64           `gorm:"column:id;type:bigint;primaryKey"`
	Name            string          `gorm:"column:name;type:varchar;size:255"`
	ServiceType     string          `gorm:"column:service_type;type:varchar;size:255"`
	Fee             decimal.Decimal `gorm:"column:fee;type:numeric(12,2)"`
	DurationMinutes int             `gorm:"column:duration_minutes;type:int"`
	Active          bool            `gorm:"column:active;type:boolean"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;type:timestamptz"`
}

func (CatalogService) TableName() string { return "catalog_service" }
