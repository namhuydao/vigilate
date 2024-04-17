package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/namhuydao/vigilate/internal/helpers"
)

// Settings displays the settings page
func (repo *DBRepo) Settings(w http.ResponseWriter, r *http.Request) {
	td := helpers.TemplateData{
		DataMap: map[string]any{
			"PageTitle": "Settings",
			"PageUrl":   "settings",
		},
	}
	helpers.HxRender(w, r, "settings", td, printTemplateError)
}

// PostSettings saves site settings
func (repo *DBRepo) PostSettings(w http.ResponseWriter, r *http.Request) {
	prefMap := make(map[string]string)

	prefMap["site_url"] = r.Form.Get("site_url")
	prefMap["notify_name"] = r.Form.Get("notify_name")
	prefMap["notify_email"] = r.Form.Get("notify_email")
	prefMap["smtp_server"] = r.Form.Get("smtp_server")
	prefMap["smtp_port"] = r.Form.Get("smtp_port")
	prefMap["smtp_user"] = r.Form.Get("smtp_user")
	prefMap["smtp_password"] = r.Form.Get("smtp_password")
	prefMap["sms_enabled"] = r.Form.Get("sms_enabled")
	prefMap["sms_provider"] = r.Form.Get("sms_provider")
	prefMap["twilio_phone_number"] = r.Form.Get("twilio_phone_number")
	prefMap["twilio_sid"] = r.Form.Get("twilio_sid")
	prefMap["twilio_auth_token"] = r.Form.Get("twilio_auth_token")
	prefMap["smtp_from_email"] = r.Form.Get("smtp_from_email")
	prefMap["smtp_from_name"] = r.Form.Get("smtp_from_name")
	prefMap["notify_via_sms"] = r.Form.Get("notify_via_sms")
	prefMap["notify_via_email"] = r.Form.Get("notify_via_email")
	prefMap["sms_notify_number"] = r.Form.Get("sms_notify_number")

	if r.Form.Get("sms_enabled") == "0" {
		prefMap["notify_via_sms"] = "0"
	}

	err := repo.DB.InsertOrUpdateSitePreferences(prefMap)
	if err != nil {
		log.Println(err)
		ClientError(w, r, http.StatusBadRequest)
		return
	}

	// update setup config
	for k, v := range prefMap {
		app.PreferenceMap[k] = v
	}

	app.Session.Put(r.Context(), "flash", "Changes saved")
	http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
}

// SetSystemPref sets a given system preference to supplied value, and returns JSON response
func (repo *DBRepo) SetSystemPref(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		PrefName  string `json:"pref_name"`
		PrefValue string `json:"pref_value"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	var resp JsonResp
	resp.Ok = true
	resp.Message = ""

	err = repo.DB.UpdateSystemPref(requestBody.PrefName, requestBody.PrefValue)
	if err != nil {
		resp.Ok = false
		resp.Message = err.Error()
	}

	repo.App.PreferenceMap["monitoring_live"] = requestBody.PrefValue

	writeJsonResponse(w, http.StatusOK, resp)
}

// ToggleMonitoring turns monitoring on and off
func (repo *DBRepo) ToggleMonitoring(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		Enabled int `json:"enabled"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	if requestBody.Enabled == 1 {
		// start monitoring
		repo.App.PreferenceMap["monitoring_live"] = "1"
		repo.StartMonitoring()
		repo.App.Scheduler.Start()
	} else {
		// stop monitoring
		repo.App.PreferenceMap["monitoring_live"] = "0"

		// remove all items in map from schedule
		for _, x := range repo.App.MonitorMap {
			repo.App.Scheduler.Remove(x)
		}

		// empty the map
		for k := range repo.App.MonitorMap {
			delete(repo.App.MonitorMap, k)
		}

		// delete all entries from schedule, to be sure
		for _, i := range repo.App.Scheduler.Entries() {
			repo.App.Scheduler.Remove(i.ID)
		}

		repo.App.Scheduler.Stop()

		data := make(map[string]string)
		data["message"] = "Monitoring is off!"
		err = app.WsClient.Trigger("public-channel", "app-stopping", data)
		if err != nil {
			log.Println(err)
		}

	}

	var resp JsonResp
	resp.Ok = true

	writeJsonResponse(w, http.StatusOK, resp)
}
