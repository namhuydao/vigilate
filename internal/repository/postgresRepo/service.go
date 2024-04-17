package postgresRepo

import (
	"context"
	"log"
	"time"

	"github.com/namhuydao/vigilate/internal/models"
)

func (m *postgresDBRepo) GetAllServiceStatusCounts() (models.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var result models.Result

	query := `SELECT
	SUM(CASE WHEN status = 'healthy' AND active = 1 THEN 1 ELSE 0 END ) healthy,
	SUM(CASE WHEN status = 'warning' AND active = 1 THEN 1 ELSE 0 END ) warning,
	SUM(CASE WHEN status = 'problem' AND active = 1 THEN 1 ELSE 0 END ) problem,
	SUM(CASE WHEN status = 'pending' AND active = 1 THEN 1 ELSE 0 END ) pending
	FROM host_services`

	row := m.DB.QueryRowContext(ctx, query)
	err := row.Scan(
		&result.Healthy,
		&result.Warning,
		&result.Problem,
		&result.Pending,
	)
	if err != nil {
		return models.Result{}, err
	}

	return result, nil
}

func (m *postgresDBRepo) GetServiceStatusCounts(status string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var count int

	query := `SELECT COALESCE(COUNT(id), 0) total FROM host_services WHERE status = $1 AND active = 1`

	err := m.DB.QueryRowContext(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// UpdateHostServiceStatus updates the active status of a host service
func (m *postgresDBRepo) UpdateHostServiceStatus(hostID, serviceID, active int, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE host_services SET active = $1, status = $2 WHERE host_id = $3 AND service_id = $4`

	_, err := m.DB.ExecContext(ctx, stmt, active, status, hostID, serviceID)
	if err != nil {
		return err
	}
	return nil
}

// UpdateHostService updates a host service in the database
func (m *postgresDBRepo) UpdateHostService(hs models.HostService) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE
   			host_services SET
   				host_id = $1, service_id = $2, active = $3,
				  	schedule_number = $4, schedule_unit = $5,
				  	last_check = $6, status = $7, updated_at = $8, last_message = $9
				WHERE
					id = $10`

	_, err := m.DB.ExecContext(ctx, stmt,
		hs.HostID,
		hs.ServiceID,
		hs.Active,
		hs.ScheduleNumber,
		hs.ScheduleUnit,
		hs.LastCheck,
		hs.Status,
		hs.UpdatedAt,
		hs.LastMessage,
		hs.ID,
	)
	if err != nil {
		return err
	}
	return nil
}

// GetServicesByStatus returns all active services with a given status
func (m *postgresDBRepo) GetServicesByStatus(status string) ([]models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT
			hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number, hs.schedule_unit,
			hs.last_check, hs.status, hs.created_at, hs.updated_at,
			h.host_name, s.service_name, hs.last_message
		FROM
			host_services hs
			LEFT JOIN hosts h ON (hs.host_id = h.id)
			LEFT JOIN services s ON (hs.service_id = s.id)
		WHERE
			status = $1
			AND hs.active = 1
		ORDER BY
			 host_name, service_name`

	var services []models.HostService

	rows, err := m.DB.QueryContext(ctx, query, status)
	if err != nil {
		return services, err
	}
	defer rows.Close()

	for rows.Next() {
		var h models.HostService

		err := rows.Scan(
			&h.ID,
			&h.HostID,
			&h.ServiceID,
			&h.Active,
			&h.ScheduleNumber,
			&h.ScheduleUnit,
			&h.LastCheck,
			&h.Status,
			&h.CreatedAt,
			&h.UpdatedAt,
			&h.HostName,
			&h.Service.ServiceName,
			&h.LastMessage,
		)
		if err != nil {
			return nil, err
		}

		services = append(services, h)
	}

	return services, nil
}

// GetHostServiceByID gets a host service by id
func (m *postgresDBRepo) GetHostServiceByID(id int) (models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number,
			hs.schedule_unit, hs.last_check, hs.status, hs.created_at, hs.updated_at,
			s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at, h.host_name,
		    hs.last_message

		FROM host_services hs
		LEFT JOIN services s ON (hs.service_id = s.id)
		LEFT JOIN hosts h ON (hs.host_id = h.id)

		WHERE hs.id = $1
`

	var hs models.HostService

	row := m.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&hs.ID,
		&hs.HostID,
		&hs.ServiceID,
		&hs.Active,
		&hs.ScheduleNumber,
		&hs.ScheduleUnit,
		&hs.LastCheck,
		&hs.Status,
		&hs.CreatedAt,
		&hs.UpdatedAt,
		&hs.Service.ID,
		&hs.Service.ServiceName,
		&hs.Service.Active,
		&hs.Service.Icon,
		&hs.Service.CreatedAt,
		&hs.Service.UpdatedAt,
		&hs.HostName,
		&hs.LastMessage,
	)

	if err != nil {
		log.Println(err)
		return hs, err
	}

	return hs, nil
}

// GetServicesToMonitor gets all host services we want to monitor
func (m *postgresDBRepo) GetServicesToMonitor() ([]models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number,
			hs.schedule_unit, hs.last_check, hs.status, hs.created_at, hs.updated_at,
			s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at,
			h.host_name, hs.last_message
		FROM
		     host_services hs
			LEFT JOIN services s ON (hs.service_id = s.id)
			LEFT JOIN hosts h ON (h.id = hs.host_id)
		WHERE
			h.active = 1
			AND hs.active = 1`

	var services []models.HostService

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()

	for rows.Next() {
		var h models.HostService
		err = rows.Scan(
			&h.ID,
			&h.HostID,
			&h.ServiceID,
			&h.Active,
			&h.ScheduleNumber,
			&h.ScheduleUnit,
			&h.LastCheck,
			&h.Status,
			&h.CreatedAt,
			&h.UpdatedAt,
			&h.Service.ID,
			&h.Service.ServiceName,
			&h.Service.Active,
			&h.Service.Icon,
			&h.Service.CreatedAt,
			&h.Service.UpdatedAt,
			&h.HostName,
			&h.LastMessage,
		)
		if err != nil {
			log.Println(err)
			return services, err
		}
		services = append(services, h)
	}

	return services, nil
}

// GetHostServiceByHostIDServiceID gets a host service by host id and service id
func (m *postgresDBRepo) GetHostServiceByHostIDServiceID(hostID, serviceID int) (models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number,
			hs.schedule_unit, hs.last_check, hs.status, hs.created_at, hs.updated_at,
			s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at, h.host_name,
		    hs.last_message

		FROM host_services hs
		LEFT JOIN services s ON (hs.service_id = s.id)
		LEFT JOIN hosts h ON (hs.host_id = h.id)

		WHERE hs.host_id = $1 AND hs.service_id = $2
		`

	var hs models.HostService

	row := m.DB.QueryRowContext(ctx, query, hostID, serviceID)

	err := row.Scan(
		&hs.ID,
		&hs.HostID,
		&hs.ServiceID,
		&hs.Active,
		&hs.ScheduleNumber,
		&hs.ScheduleUnit,
		&hs.LastCheck,
		&hs.Status,
		&hs.CreatedAt,
		&hs.UpdatedAt,
		&hs.Service.ID,
		&hs.Service.ServiceName,
		&hs.Service.Active,
		&hs.Service.Icon,
		&hs.Service.CreatedAt,
		&hs.Service.UpdatedAt,
		&hs.HostName,
		&hs.LastMessage,
	)

	if err != nil {
		log.Println(err)
		return hs, err
	}

	return hs, nil
}

func (m *postgresDBRepo) GetCountHostServiceActive(id int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var count int

	query := `SELECT COALESCE(SUM(id), 0) total FROM host_services WHERE active = 1 AND id = $1`

	err := m.DB.QueryRowContext(ctx, query, id).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
