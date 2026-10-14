package repository

import (
	"context"
	"errors"
	"fmt"
	"scheduler-service/internal/adapters/database/models"
	database_provider "scheduler-service/internal/adapters/database/provider"
	"scheduler-service/internal/constants"
	"scheduler-service/internal/domain/entity"
	"scheduler-service/internal/domain/ports"
	"time"

	"gorm.io/gorm"
)

// hubColumns selects opening hours as "HH:MM" text, which parseHour reads,
// rather than relying on how the driver decodes Postgres's time type.
const hubColumns = `hub.id, hub.name, hub.address, hub.timezone, hub.active, hub.updated_at,
	to_char(hub.open_hour, 'HH24:MI') AS open_hour,
	to_char(hub.close_hour, 'HH24:MI') AS close_hour`

type CatalogRepository struct {
	db *gorm.DB
}

func NewCatalogRepository(db *gorm.DB) ports.CatalogRepository {
	return CatalogRepository{db: db}
}

// ListHubs implements [ports.CatalogRepository].
func (r CatalogRepository) ListHubs(ctx context.Context) ([]entity.Hub, error) {
	var rows []models.Hub
	err := database_provider.DBFromContext(ctx, r.db).
		Table("hub").
		Select(hubColumns+`, (SELECT count(*) FROM hub_bay b WHERE b.hub_id = hub.id AND b.status = ?) AS bay_count`, constants.BayAvailable).
		Where("hub.active").
		Order("hub.name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	hubs := make([]entity.Hub, 0, len(rows))
	for _, row := range rows {
		hub, err := toHub(row)
		if err != nil {
			return nil, err
		}
		hubs = append(hubs, hub)
	}
	return hubs, nil
}

// GetHub implements [ports.CatalogRepository].
func (r CatalogRepository) GetHub(ctx context.Context, id int64) (entity.Hub, error) {
	var rows []models.Hub
	err := database_provider.DBFromContext(ctx, r.db).
		Table("hub").
		Select(hubColumns).
		Where("hub.id = ? AND hub.active", id).
		Limit(1).
		Scan(&rows).Error
	if err != nil {
		return entity.Hub{}, err
	}
	if len(rows) == 0 {
		return entity.Hub{}, fmt.Errorf("%w: hub %d", ports.ErrNotFound, id)
	}
	return toHub(rows[0])
}

// ListBookableBays implements [ports.CatalogRepository].
func (r CatalogRepository) ListBookableBays(ctx context.Context, hubID int64) ([]entity.Bay, error) {
	var rows []models.HubBay
	err := database_provider.DBFromContext(ctx, r.db).
		Where("hub_id = ? AND status = ?", hubID, constants.BayAvailable).
		Order("code").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	bays := make([]entity.Bay, len(rows))
	for i, row := range rows {
		bays[i] = entity.Bay{ID: row.ID, HubID: row.HubID, Code: row.Code, Status: row.Status}
	}
	return bays, nil
}

// ListServices implements [ports.CatalogRepository].
func (r CatalogRepository) ListServices(ctx context.Context) ([]entity.CatalogService, error) {
	var rows []models.CatalogService
	err := database_provider.DBFromContext(ctx, r.db).
		Where("active").
		Order("service_type, name").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	services := make([]entity.CatalogService, len(rows))
	for i, row := range rows {
		services[i] = entity.CatalogService{
			ID:              row.ID,
			Name:            row.Name,
			ServiceType:     row.ServiceType,
			Fee:             row.Fee,
			DurationMinutes: row.DurationMinutes,
			Active:          row.Active,
		}
	}
	return services, nil
}

func toHub(row models.Hub) (entity.Hub, error) {
	open, err := parseHour(row.OpenHour)
	if err != nil {
		return entity.Hub{}, err
	}
	closing, err := parseHour(row.CloseHour)
	if err != nil {
		return entity.Hub{}, err
	}
	return entity.Hub{
		ID:        row.ID,
		Name:      row.Name,
		Address:   row.Address,
		OpenHour:  open,
		CloseHour: closing,
		TimeZone:  row.TimeZone,
		Active:    row.Active,
		BayCount:  row.BayCount,
	}, nil
}

// parseHour turns "HH:MM" into an offset from midnight.
func parseHour(value string) (time.Duration, error) {
	t, err := time.Parse("15:04", value)
	if err != nil {
		return 0, errors.Join(fmt.Errorf("hub opening hour %q", value), err)
	}
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute, nil
}
