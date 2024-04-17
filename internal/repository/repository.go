package repository

import "github.com/namhuydao/vigilate/internal/models"

// DatabaseRepo is the database repository
type DatabaseRepo interface {
	// Preferences
	AllPreferences() ([]models.Preference, error)
	SetSystemPref(name, value string) error
	UpdateSystemPref(name, value string) error
	InsertOrUpdateSitePreferences(pm map[string]string) error

	// Users
	AllUsers() ([]models.User, error)
	GetUserById(id int) (models.User, error)
	InsertUser(u models.User) (int, error)
	UpdateUser(u models.User) error
	DeleteUser(id int) error
	UpdatePassword(id int, newPassword string) error

	// Authentication
	Authenticate(email, testPassword string) (int, error)
	InsertRememberMeToken(id int, token string) error
	DeleteRememberMeToken(token string) error
	CheckForRememberMeToken(id int, token string) bool

	// Hosts
	InsertHost(h models.Host) (int, error)
	GetHostByID(id int) (models.Host, error)
	UpdateHost(h models.Host) error
	AllHosts() ([]models.Host, error)
	UpdateHostServiceStatus(hostID, serviceID, active int, status string) error
	GetAllServiceStatusCounts() (models.Result, error)
	GetServiceStatusCounts(status string) (int, error)
	GetServicesByStatus(status string) ([]models.HostService, error)
	GetCountHostServiceActive(id int) (int, error)
	GetHostServiceByID(id int) (models.HostService, error)
	GetHostServiceByHostIDServiceID(hostID, serviceID int) (models.HostService, error)
	UpdateHostService(hs models.HostService) error
	GetServicesToMonitor() ([]models.HostService, error)
	GetAllEvents() ([]models.Event, error)
	InsertEvent(e models.Event) error
}
