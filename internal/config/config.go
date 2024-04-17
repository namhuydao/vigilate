package config

import (
	"html/template"

	"github.com/namhuydao/vigilate/internal/driver"
	"github.com/namhuydao/vigilate/internal/models"

	"github.com/alexedwards/scs/v2"
	"github.com/robfig/cron/v3"
)

// MailData holds info for sending an email
type MailData struct {
	ToName       string
	ToAddress    string
	FromName     string
	FromAddress  string
	AdditionalTo []string
	Subject      string
	Content      template.HTML
	Template     string
	CC           []string
	UseHermes    bool
	Attachments  []string
	StringMap    map[string]string
	IntMap       map[string]int
	FloatMap     map[string]float32
	RowSets      map[string]interface{}
}

// MailJob is the unit of work to be performed when sending an email to chan
type MailJob struct {
	MailMessage MailData
}

// AppConfig holds application configuration
type AppConfig struct {
	DB            *driver.DB
	Session       *scs.SessionManager
	InProduction  bool
	Domain        string
	MonitorMap    map[int]cron.EntryID
	PreferenceMap map[string]string
	Scheduler     *cron.Cron
	WsClient      models.WSClient
	PusherSecret  string
	TemplateCache map[string]*template.Template
	MailQueue     chan MailJob
	Version       string
	Identifier    string
}

var KnownRoutes = map[string]bool{}
