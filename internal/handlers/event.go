package handlers

import (
	"log"
	"net/http"

	"github.com/namhuydao/vigilate/internal/helpers"
)

// Events displays the events page
func (repo *DBRepo) Events(w http.ResponseWriter, r *http.Request) {
	events, err := repo.DB.GetAllEvents()
	if err != nil {
		log.Println(err)
		return
	}

	td := helpers.TemplateData{
		DataMap: map[string]any{
			"events":    events,
			"PageTitle": "Events",
			"PageUrl":   "events",
		},
	}

	helpers.HxRender(w, r, "events", td, printTemplateError)
}
