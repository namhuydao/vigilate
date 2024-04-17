package handlers

import (
	"log"
	"net/http"

	"github.com/namhuydao/vigilate/internal/helpers"
)

func (repo *DBRepo) AdminOverview(w http.ResponseWriter, r *http.Request) {
	td := helpers.TemplateData{
		DataMap: map[string]any{
			"PageTitle": "Dashboard",
			"PageUrl":   "dashboard",
		},
	}
	helpers.HxRender(w, r, "layout", td, printTemplateError)
}

// AdminDashboard displays the dashboard
func (repo *DBRepo) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	result, err := repo.DB.GetAllServiceStatusCounts()

	if err != nil {
		log.Println(err)
		return
	}

	allHosts, err := repo.DB.AllHosts()
	if err != nil {
		log.Println(err)
		return
	}

	td := helpers.TemplateData{
		DataMap: map[string]any{
			"NoHealthy": result.Healthy,
			"NoProblem": result.Problem,
			"NoPending": result.Pending,
			"NoWarning": result.Warning,
			"Hosts":     allHosts,
			"PageTitle": "Dashboard",
			"PageUrl":   "dashboard",
		},
	}

	helpers.HxRender(w, r, "dashboard", td, printTemplateError)
}
