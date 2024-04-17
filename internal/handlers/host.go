package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/namhuydao/vigilate/internal/helpers"
	"github.com/namhuydao/vigilate/internal/models"

	"github.com/go-chi/chi/v5"
)

// AllHosts displays list of all hosts
func (repo *DBRepo) AllHosts(w http.ResponseWriter, r *http.Request) {
	// get all hosts from database
	hosts, err := repo.DB.AllHosts()
	if err != nil {
		log.Println(err)
		return
	}

	td := helpers.TemplateData{
		DataMap: map[string]any{
			"hosts":     hosts,
			"PageTitle": "Hosts",
			"PageUrl":   "host/all",
		},
	}

	helpers.HxRender(w, r, "hosts", td, printTemplateError)
}

// Host shows the host add/edit form
func (repo *DBRepo) Host(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	activeTab := r.URL.Query().Get("activeTab")

	var h models.Host

	if id > 0 {
		// get the host from the database
		host, err := repo.DB.GetHostByID(id)
		if err != nil {
			log.Println(err)
			return
		}
		h = host
	}

	td := helpers.TemplateData{
		DataMap: map[string]any{
			"host":      h,
			"PageTitle": "Host",
			"PageUrl":   fmt.Sprintf("host/%d", h.ID),
			"ActiveTab": activeTab,
		},
	}

	helpers.HxRender(w, r, "host", td, printTemplateError)
}

// PostHost handles posting of host form
func (repo *DBRepo) PostHost(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	var h models.Host

	if id > 0 {
		// get the host from the database
		host, err := repo.DB.GetHostByID(id)
		if err != nil {
			log.Println(err)
			return
		}
		h = host
	}

	h.HostName = r.Form.Get("host_name")
	h.CanonicalName = r.Form.Get("canonical_name")
	h.URL = r.Form.Get("url")
	h.IP = r.Form.Get("ip")
	h.IPV6 = r.Form.Get("ipv6")
	h.Location = r.Form.Get("location")
	h.OS = r.Form.Get("os")
	active, _ := strconv.Atoi(r.Form.Get("active"))
	h.Active = active

	if id > 0 {
		err := repo.DB.UpdateHost(h)
		if err != nil {
			log.Println(err)
			return
		}
	} else {
		newID, err := repo.DB.InsertHost(h)
		if err != nil {
			log.Println(err)
			helpers.ServerError(w, r, err)
			return
		}
		h.ID = newID
	}

	repo.App.Session.Put(r.Context(), "flash", "Changes saved")
	http.Redirect(w, r, fmt.Sprintf("/admin/host/%d", h.ID), http.StatusSeeOther)
}

// ToggleServiceForHost turns a host service on or off (active or inactive)
func (repo *DBRepo) ToggleServiceForHost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
	}

	var resp = JsonResp{Ok: true}

	hostID, _ := strconv.Atoi(r.Form.Get("host_id"))
	serviceID, _ := strconv.Atoi(r.Form.Get("service_id"))
	active, _ := strconv.Atoi(r.Form.Get("active"))
	status := r.Form.Get("status")
	log.Println(status)

	err = repo.DB.UpdateHostServiceStatus(hostID, serviceID, active, status)
	if err != nil {
		log.Println(err)
		resp.Ok = false
	}

	// broadcast
	hs, _ := repo.DB.GetHostServiceByHostIDServiceID(hostID, serviceID)
	h, _ := repo.DB.GetHostByID(hostID)

	// add or remove from schedule
	repo.PushStatusChangeEvent(h, hs, "pending")
	repo.updateHostServiceStatusCount(hs, "pending", "")
	if active == 1 {
		repo.PushScheduleChangeEvent(hs, "pending")
		repo.AddToMonitorMap(hs)
	} else {
		repo.RemoveFromMonitorMap(hs)
	}

	writeJsonResponse(w, http.StatusOK, resp)
}
