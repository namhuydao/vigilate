package handlers

import (
	"fmt"
	"github.com/namhuydao/vigilate/internal/helpers"
	"github.com/namhuydao/vigilate/internal/models"
	"log"
	"net/http"
	"sort"
)

// ByHost allows us to sort by host
type ByHost []models.Schedule

// Len is used to sort by host
func (a ByHost) Len() int { return len(a) }

// Less is used to sort by host
func (a ByHost) Less(i, j int) bool { return a[i].Host < a[j].Host }

// Swap is used to sort by host
func (a ByHost) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// ListEntries lists schedule entries
func (repo *DBRepo) ListEntries(w http.ResponseWriter, r *http.Request) {
	var items []models.Schedule
	if len(repo.App.MonitorMap) > 0 {
		for k, v := range repo.App.MonitorMap {
			countActive, err := repo.DB.GetCountHostServiceActive(k)
			if err != nil {
				log.Println(err)
				return
			}
			log.Println(countActive)

			hs, err := repo.DB.GetHostServiceByID(k)
			if err != nil {
				log.Println(err)
				return
			}
			if countActive > 0 {
				var item models.Schedule
				item.ID = k
				item.EntryID = v
				item.Entry = app.Scheduler.Entry(v)
				item.ScheduleText = fmt.Sprintf("@every %d%s", hs.ScheduleNumber, hs.ScheduleUnit)
				item.LastRunFromHS = hs.LastCheck
				item.Host = hs.HostName
				item.Service = hs.Service.ServiceName
				items = append(items, item)

			} else {
				continue
			}
			sort.Sort(ByHost(items))
		}
	}

	td := helpers.TemplateData{
		DataMap: map[string]any{
			"items":     items,
			"PageTitle": "Schedule",
			"PageUrl":   "schedule",
		},
	}

	helpers.HxRender(w, r, "schedule", td, printTemplateError)
}
