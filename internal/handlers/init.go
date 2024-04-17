package handlers

import (
	"encoding/json"
	"github.com/namhuydao/vigilate/internal/config"
	"github.com/namhuydao/vigilate/internal/driver"
	"github.com/namhuydao/vigilate/internal/repository"
	"github.com/namhuydao/vigilate/internal/repository/postgresRepo"
	"net/http"
	"time"
)

// Repo is the repository
var Repo *DBRepo
var app *config.AppConfig

// DBRepo is the db repo
type DBRepo struct {
	App *config.AppConfig
	DB  repository.DatabaseRepo
}

// NewHandlers creates the handlers
func NewHandlers(repo *DBRepo, a *config.AppConfig) {
	Repo = repo
	app = a
}

// NewDBHandlers creates db repo for postgres
func NewDBHandlers(db *driver.DB, a *config.AppConfig) *DBRepo {
	return &DBRepo{
		App: a,
		DB:  postgresRepo.NewPostgresRepo(db.SQL, a),
	}
}

const (
	HTTP           = 1
	HTTPS          = 2
	SSLCertificate = 3
)

type JsonResp struct {
	Ok            bool      `json:"ok"`
	Message       string    `json:"message"`
	ServiceId     int       `json:"service_id"`
	HostServiceId int       `json:"host_service_id"`
	HostId        int       `json:"host_id"`
	OldStatus     string    `json:"old_status"`
	NewStatus     string    `json:"new_status"`
	LastCheck     time.Time `json:"last_check"`
}

func writeJsonResponse(w http.ResponseWriter, statusCode int, resp JsonResp) {
	out, _ := json.MarshalIndent(resp, "", "   ")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err := w.Write(out)
	if err != nil {
		return
	}
}
