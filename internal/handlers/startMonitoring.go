package handlers

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

type job struct {
	HostServiceId int
}

func (j job) Run() {
	Repo.ScheduledCheck(j.HostServiceId)
}

func (repo *DBRepo) StartMonitoring() {
	data := make(map[string]string)
	data["message"] = "Monitoring is live..."

	err := app.WsClient.Trigger("public-channel", "app-starting", data)
	if err != nil {
		log.Println(err)
	}

	servicesToMonitor, err := repo.DB.GetServicesToMonitor()
	if err != nil {
		log.Println(err)
	}

	for _, service := range servicesToMonitor {
		var schedule string
		if service.ScheduleUnit == "d" {
			schedule = fmt.Sprintf("@every %d%s", service.ScheduleNumber*24, "h")
		} else {
			schedule = fmt.Sprintf("@every %d%s", service.ScheduleNumber, service.ScheduleUnit)
		}

		var j job
		j.HostServiceId = service.ID
		schedulerId, err := app.Scheduler.AddJob(schedule, j)
		if err != nil {
			log.Println(err)
		}

		app.MonitorMap[service.ID] = schedulerId

		payload := make(map[string]string)
		payload["message"] = fmt.Sprintf("Monitoring is running... %d", service.ScheduleNumber)
		payload["host_service_id"] = strconv.Itoa(service.ID)
		yearOne := time.Date(0001, 11, 17, 20, 34, 58, 65138737, time.UTC)

		if app.Scheduler.Entry(app.MonitorMap[service.ID]).Next.After(yearOne) {
			payload["next_run"] = app.Scheduler.Entry(app.MonitorMap[service.ID]).Next.Format("2006-01-02 15:04:05")
		} else {
			payload["next_run"] = "Pending..."
		}
		payload["host"] = service.HostName
		payload["service"] = service.Service.ServiceName
		if service.LastCheck.After(yearOne) {
			payload["last_run"] = service.LastCheck.Format("2006-01-02 15:04:05")
		} else {
			payload["last_run"] = "Pending..."
		}
		payload["schedule"] = fmt.Sprintf("@every %d%s", service.ScheduleNumber, service.ScheduleUnit)

		err = app.WsClient.Trigger("public-channel", "next-run-event", payload)
		if err != nil {
			log.Println(err)
		}

		err = app.WsClient.Trigger("public-channel", "schedule-changed-event", payload)
		if err != nil {
			log.Println(err)
		}
	}

}
