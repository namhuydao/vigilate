package handlers

import (
	"fmt"
	"github.com/namhuydao/vigilate/internal/certificateutils"
	"github.com/namhuydao/vigilate/internal/config"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/namhuydao/vigilate/internal/helpers"
	"github.com/namhuydao/vigilate/internal/models"
	"github.com/namhuydao/vigilate/internal/sms"

	"github.com/go-chi/chi/v5"
)

func (repo *DBRepo) ScheduledCheck(hostServiceId int) {
	hs, err := Repo.DB.GetHostServiceByID(hostServiceId)
	if err != nil {
		log.Println(err)
		return
	}

	host, err := Repo.DB.GetHostByID(hs.HostID)
	if err != nil {
		log.Println(err)
		return
	}

	// test the service
	newStatus, msg := Repo.testServiceForHost(host, hs)

	if newStatus != hs.Status {
		repo.updateHostServiceStatusCount(hs, newStatus, msg)
	}

}

func (repo *DBRepo) updateHostServiceStatusCount(hs models.HostService, newStatus, msg string) {
	hs.Status = newStatus
	hs.LastMessage = msg
	hs.LastCheck = time.Now()

	err := Repo.DB.UpdateHostService(hs)
	if err != nil {
		log.Println(err)
		return
	}

	counts, err := repo.DB.GetAllServiceStatusCounts()
	log.Println(counts)

	if err != nil {
		log.Println(err)
		return
	}

	data := make(map[string]string)
	data["healthy_count"] = strconv.Itoa(counts.Healthy)
	data["pending_count"] = strconv.Itoa(counts.Pending)
	data["problem_count"] = strconv.Itoa(counts.Problem)
	data["warning_count"] = strconv.Itoa(counts.Warning)

	repo.BroadcastMessage("public-channel", "host-service-count-changed", data)
}

func (repo *DBRepo) BroadcastMessage(channel, messageType string, data map[string]string) {
	err := app.WsClient.Trigger(channel, messageType, data)
	if err != nil {
		log.Println(err)
	}
}

func (repo *DBRepo) TestCheck(w http.ResponseWriter, req *http.Request) {
	hostServiceId, _ := strconv.Atoi(chi.URLParam(req, "id"))
	oldStatus := chi.URLParam(req, "oldStatus")

	ok := true

	hs, err := repo.DB.GetHostServiceByID(hostServiceId)
	if err != nil {
		log.Println(err)
		ok = false
	}

	h, err := repo.DB.GetHostByID(hs.HostID)
	if err != nil {
		log.Println(err)
		ok = false
	}

	newStatus, msg := repo.testServiceForHost(h, hs)
	event := models.Event{
		EventType:     newStatus,
		HostServiceID: hs.ID,
		HostID:        h.ID,
		ServiceName:   hs.Service.ServiceName,
		HostName:      h.HostName,
		Message:       msg,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = Repo.DB.InsertEvent(event)
	if err != nil {
		log.Println(err)
	}

	if newStatus != hs.Status {
		repo.PushStatusChangeEvent(h, hs, newStatus)
	}

	hs.Status = newStatus
	hs.LastMessage = msg
	hs.LastCheck = time.Now()
	hs.UpdatedAt = time.Now()
	err = repo.DB.UpdateHostService(hs)
	if err != nil {
		log.Println(err)
		ok = false
	}

	var resp JsonResp
	if ok {
		resp = JsonResp{
			Ok:            true,
			Message:       msg,
			ServiceId:     hs.ServiceID,
			HostId:        hs.HostID,
			HostServiceId: hs.ID,
			OldStatus:     oldStatus,
			NewStatus:     newStatus,
			LastCheck:     time.Now(),
		}
	} else {
		resp.Ok = false
		resp.Message = "Something went wrong"
	}

	writeJsonResponse(w, http.StatusOK, resp)
}

func (repo *DBRepo) testServiceForHost(h models.Host, hs models.HostService) (string, string) {
	var msg, newStatus string

	switch hs.ServiceID {
	case HTTP:
		msg, newStatus = testHTTPForHost(h.URL)
		break
	case HTTPS:
		msg, newStatus = testHTTPSForHost(h.URL)
		break
	case SSLCertificate:
		msg, newStatus = testSSLForHost(h.URL)
	}

	if hs.Status != newStatus {
		repo.PushStatusChangeEvent(h, hs, newStatus)
		event := models.Event{
			EventType:     newStatus,
			HostServiceID: hs.ID,
			HostID:        h.ID,
			ServiceName:   hs.Service.ServiceName,
			HostName:      h.HostName,
			Message:       msg,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := Repo.DB.InsertEvent(event)
		if err != nil {
			log.Println(err)
		}

		if repo.App.PreferenceMap["notify_via_email"] == "1" {
			if hs.Status != "pending" {
				mailMsg := config.MailData{
					ToName:    repo.App.PreferenceMap["notify_name"],
					ToAddress: repo.App.PreferenceMap["notify_email"],
				}

				if newStatus == "healthy" {
					mailMsg.Subject = fmt.Sprintf("HEALTHY: service %s on %s", hs.Service.ServiceName, hs.HostName)
					mailMsg.Content = template.HTML(fmt.Sprintf(`<p>Service %s on %s reported healthy status</p>
						<p><strong>Message received: %s</strong></p>
						`, hs.Service.ServiceName, hs.HostName, msg))
				} else if newStatus == "problem" {
					mailMsg.Subject = fmt.Sprintf("PROBLEM: service %s on %s", hs.Service.ServiceName, hs.HostName)
					mailMsg.Content = template.HTML(fmt.Sprintf(`<p>Service %s on %s reported problem</p>
						<p><strong>Message received: %s</strong></p>
						`, hs.Service.ServiceName, hs.HostName, msg))
				} else if newStatus == "warning" {

				}
				helpers.SendEmail(mailMsg)
			}
		}

		if repo.App.PreferenceMap["notify_via_sms"] == "1" {
			to := repo.App.PreferenceMap["sms_notify_number"]
			smsMessage := ""

			if newStatus == "healthy" {
				smsMessage = fmt.Sprintf("HEALTHY: service %s on %s", hs.Service.ServiceName, hs.HostName)
			} else if newStatus == "problem" {
				smsMessage = fmt.Sprintf("PROBLEM: service %s on %s", hs.Service.ServiceName, hs.HostName)
			} else if newStatus == "warning" {
				smsMessage = fmt.Sprintf("WARNING: service %s on %s", hs.Service.ServiceName, hs.HostName)
			}

			err = sms.SendTextTwilio(to, smsMessage, repo.App)
			if err != nil {
				log.Println(err)
			}
		}

	}

	repo.PushScheduleChangeEvent(hs, newStatus)

	return newStatus, msg
}

func (repo *DBRepo) PushStatusChangeEvent(h models.Host, hs models.HostService, newStatus string) {
	count, err := repo.DB.GetServiceStatusCounts(hs.Status)
	if err != nil {
		log.Println(err)
	}

	data := make(map[string]string)
	data["host_id"] = strconv.Itoa(h.ID)
	data["host_service_id"] = strconv.Itoa(hs.ID)
	data["host_name"] = h.HostName
	data["service_name"] = hs.Service.ServiceName
	data["icon"] = hs.Service.Icon
	data["status"] = newStatus
	data["message"] = fmt.Sprintf("%s on %s reports %s", hs.Service.ServiceName, h.HostName, newStatus)
	data["last_check"] = time.Now().Format("2006-01-02 15:04:05")
	data["total_new_status"] = strconv.Itoa(count - 1)

	repo.BroadcastMessage("public-channel", "host-service-status-changed", data)
}

func (repo *DBRepo) PushScheduleChangeEvent(hs models.HostService, newStatus string) {
	yearOne := time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC)
	data := make(map[string]string)
	data["host_service_id"] = strconv.Itoa(hs.ID)
	data["service_id"] = strconv.Itoa(hs.ServiceID)
	data["host_id"] = strconv.Itoa(hs.HostID)

	if app.Scheduler.Entry(repo.App.MonitorMap[hs.ID]).Next.After(yearOne) {
		data["next_run"] = repo.App.Scheduler.Entry(repo.App.MonitorMap[hs.ID]).Next.Format("2006-01-02 15:04:05")
	} else {
		data["next_run"] = "Pending..."
	}

	data["last_run"] = time.Now().Format("2006-01-02 15:04:05")
	data["host"] = hs.HostName
	data["service"] = hs.Service.ServiceName
	data["schedule"] = fmt.Sprintf("@every %d%s", hs.ScheduleNumber, hs.ScheduleUnit)
	data["status"] = newStatus
	data["icon"] = hs.Service.Icon
	repo.BroadcastMessage("public-channel", "schedule-changed-event", data)
}

func testHTTPForHost(url string) (string, string) {
	if strings.HasSuffix(url, "/") {
		strings.TrimSuffix(url, "/")
	}

	url = strings.Replace(url, "https://", "http://", -1)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("%s - %s", url, "error connecting"), "problem"
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("%s - %s", url, resp.Status), "problem"
	}
	return fmt.Sprintf("%s - %s", url, resp.Status), "healthy"
}

func testHTTPSForHost(url string) (string, string) {
	if strings.HasSuffix(url, "/") {
		strings.TrimSuffix(url, "/")
	}

	url = strings.Replace(url, "http://", "https://", -1)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("%s - %s", url, "error connecting"), "problem"
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("%s - %s", url, resp.Status), "problem"
	}
	return fmt.Sprintf("%s - %s", url, resp.Status), "healthy"
}
func scanHost(hostname string, certDetailsChannel chan certificateutils.CertificateDetails, errorsChannel chan error) {
	res, err := certificateutils.GetCertificateDetails(hostname, 10)
	if err != nil {
		errorsChannel <- err
	} else {
		certDetailsChannel <- res
	}
	return
}

func testSSLForHost(url string) (string, string) {
	if strings.HasPrefix(url, "https://") {
		url = strings.Replace(url, "https://", "", -1)
	}

	if strings.HasPrefix(url, "http://") {
		url = strings.Replace(url, "http://", "", -1)
	}

	var certDetailsChannel chan certificateutils.CertificateDetails
	var errorsChannel chan error
	certDetailsChannel = make(chan certificateutils.CertificateDetails, 1)
	errorsChannel = make(chan error, 1)

	var msg, newStatus string
	scanHost(url, certDetailsChannel, errorsChannel)

	for i, certDetailsInQueue := 0, len(certDetailsChannel); i < certDetailsInQueue; i++ {
		certDetails := <-certDetailsChannel
		certificateutils.CheckExpirationStatus(&certDetails, 30)

		if certDetails.Expired {
			// cert expired
			msg = certDetails.Hostname + " has expired!"

		} else if certDetails.ExpiringSoon {
			// cert expiring sono
			if certDetails.DaysUntilExpiration < 7 {
				msg = certDetails.Hostname + " expiring in " + strconv.Itoa(certDetails.DaysUntilExpiration) + " days"
				newStatus = "problem"
			} else {
				msg = certDetails.Hostname + " expiring in " + strconv.Itoa(certDetails.DaysUntilExpiration) + " days"
				newStatus = "warning"
			}
		} else {
			// cert okay
			msg = certDetails.Hostname + " expiring in " + strconv.Itoa(certDetails.DaysUntilExpiration) + " days"
			newStatus = "healthy"
		}
	}

	return msg, newStatus
}

func (repo *DBRepo) AddToMonitorMap(hs models.HostService) {
	if repo.App.PreferenceMap["monitoring_live"] == "1" {
		var j job
		j.HostServiceId = hs.ID
		scheduleId, err := repo.App.Scheduler.AddJob(fmt.Sprintf("@every %d%s", hs.ScheduleNumber, hs.ScheduleUnit), j)
		if err != nil {
			log.Println(err)
		}

		repo.App.MonitorMap[hs.ID] = scheduleId
		data := make(map[string]string)
		data["host_service_id"] = strconv.Itoa(hs.ID)
		data["host"] = hs.HostName
		data["service"] = hs.Service.ServiceName
		data["schedule"] = fmt.Sprintf("@every %d%s", hs.ScheduleNumber, hs.ScheduleUnit)
		data["message"] = "scheduling"
		data["next_run"] = "Pending..."
		data["last_run"] = hs.LastCheck.Format("2006-01-02 15:04:05")

		repo.BroadcastMessage("public-channel", "schedule-changed-event", data)
	}
}

func (repo *DBRepo) RemoveFromMonitorMap(hs models.HostService) {
	if repo.App.PreferenceMap["monitoring_live"] == "1" {
		repo.App.Scheduler.Remove(repo.App.MonitorMap[hs.ID])
		data := make(map[string]string)
		data["host_service_id"] = strconv.Itoa(hs.ID)

		repo.BroadcastMessage("public-channel", "schedule-item-removed-event", data)
	}
}
