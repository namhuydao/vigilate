package handlers

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/namhuydao/vigilate/internal/helpers"
	"log"
	"net/http"
)

// AllServices lists all services
func (repo *DBRepo) AllServices(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")

	services, err := repo.DB.GetServicesByStatus(status)
	if err != nil {
		log.Println(err)
		return
	}

	td := helpers.TemplateData{
		DataMap: map[string]any{
			"services":    services,
			"PageTitle":   fmt.Sprintf("All %s Services", helpers.CapitalizedString(status)),
			"PageUrl":     fmt.Sprintf("all-service-status/%s", status),
			"TableNameUp": helpers.CapitalizedString(status),
			"TableName":   status,
		},
	}

	helpers.HxRender(w, r, "serviceStatus", td, printTemplateError)
}
