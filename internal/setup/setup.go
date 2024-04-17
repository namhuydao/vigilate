package setup

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/namhuydao/vigilate/internal/middleware"
	"github.com/namhuydao/vigilate/internal/routes"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/namhuydao/vigilate/internal/config"
	"github.com/namhuydao/vigilate/internal/driver"
	"github.com/namhuydao/vigilate/internal/handlers"
	"github.com/namhuydao/vigilate/internal/helpers"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/pusher/pusher-http-go"
	"github.com/robfig/cron/v3"
)

var app config.AppConfig
var repo *handlers.DBRepo
var session *scs.SessionManager
var preferenceMap map[string]string
var wsClient pusher.Client

var vigilateVersion string
var maxWorkerPoolSize int
var maxJobMaxWorkers int

func NewSetUp(appConfig config.AppConfig, repo *handlers.DBRepo, session *scs.SessionManager,
	preferenceMap map[string]string, wsClient pusher.Client, version string,
	maxWorkerPoolSize int, maxJobMaxWorkers int) {
	app = appConfig
	repo = repo
	session = session
	preferenceMap = preferenceMap
	wsClient = wsClient
	vigilateVersion = version
	maxWorkerPoolSize = maxWorkerPoolSize
	maxJobMaxWorkers = maxJobMaxWorkers
}

func InitApp() (string, error) {
	// Get all env information
	insecurePort := os.Getenv("APP_PORT")
	identifier := os.Getenv("APP_IDENTIFIER")
	domain := os.Getenv("APP_DOMAIN")
	inProduction, err := strconv.ParseBool(os.Getenv("APP_INPRODUCTION"))
	if err != nil {
		inProduction = false
	}
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	databaseName := os.Getenv("DB_DATABASE")
	dbSsl := os.Getenv("DB_SSL")
	pusherHost := os.Getenv("PUSHER_HOST")
	pusherPort := os.Getenv("PUSHER_PORT")
	pusherApp := os.Getenv("PUSHER_APP")
	pusherKey := os.Getenv("PUSHER_KEY")
	pusherSecret := os.Getenv("PUSHER_SECRET")
	pusherSecure, err := strconv.ParseBool(os.Getenv("PUSHER_SECURE"))
	if err != nil {
		pusherSecure = false
	}

	log.Println("Connecting to database....")
	dsnString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=UTC connect_timeout=5",
		dbHost,
		dbPort,
		dbUser,
		dbPass,
		databaseName,
		dbSsl,
	)

	db, err := driver.ConnectDB(dsnString)
	if err != nil {
		log.Fatal("Cannot connect to database!", err)
	}

	// redis
	log.Printf("Initializing redis connection..")
	redisPool := &redis.Pool{
		MaxIdle: 10,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", os.Getenv("REDIS"))
		},
	}

	// session
	log.Printf("Initializing session manager....")
	session = scs.New()
	session.Store = redisstore.New(redisPool)
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.Name = fmt.Sprintf("gbsession_id_%s", identifier)
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = inProduction

	// start mail channel
	log.Println("Initializing mail channel and worker pool....")
	mailQueue := make(chan config.MailJob, maxWorkerPoolSize)

	// Start the email dispatcher
	log.Println("Starting email dispatcher....")
	dispatcher := NewDispatcher(mailQueue, maxJobMaxWorkers)
	dispatcher.run()

	// define application configuration
	a := config.AppConfig{
		DB:           db,
		Session:      session,
		InProduction: inProduction,
		Domain:       domain,
		PusherSecret: pusherSecret,
		MailQueue:    mailQueue,
		Version:      vigilateVersion,
		Identifier:   identifier,
	}

	app = a

	repo = handlers.NewDBHandlers(db, &app)
	handlers.NewHandlers(repo, &app)

	log.Println("Getting preferences...")
	preferenceMap = make(map[string]string)
	preferences, err := repo.DB.AllPreferences()
	if err != nil {
		log.Fatal("Cannot read preferences:", err)
	}

	for _, pref := range preferences {
		preferenceMap[pref.Name] = string(pref.Preference)
	}

	preferenceMap["pusher-host"] = pusherHost
	preferenceMap["pusher-port"] = pusherPort
	preferenceMap["pusherKey"] = pusherKey
	preferenceMap["identifier"] = identifier
	preferenceMap["version"] = vigilateVersion

	app.PreferenceMap = preferenceMap

	// create pusher client
	wsClient = pusher.Client{
		AppID:  pusherApp,
		Secret: pusherSecret,
		Key:    pusherKey,
		Secure: pusherSecure,
		Host:   fmt.Sprintf("%s:%s", pusherHost, pusherPort),
	}

	log.Println("Host", fmt.Sprintf("%s:%s", pusherHost, pusherPort))
	log.Println("Secure", pusherSecure)

	app.WsClient = &wsClient
	monitorMap := make(map[int]cron.EntryID)
	app.MonitorMap = monitorMap

	localZone, _ := time.LoadLocation("Local")
	scheduler := cron.New(cron.WithLocation(localZone), cron.WithChain(
		cron.DelayIfStillRunning(cron.DefaultLogger),
		cron.Recover(cron.DefaultLogger),
	))

	app.Scheduler = scheduler

	if app.PreferenceMap["monitoring_live"] == "1" {
		go handlers.Repo.StartMonitoring()
		app.Scheduler.Start()
	}

	helpers.NewHelpers(&app)
	middleware.NewMiddleware(&app, db)

	return insecurePort, err
}

func Start(port string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	ch := make(chan error)

	// create http server
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           routes.Routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	log.Printf("Starting HTTP server on port %s....", port)

	// start the server
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			ch <- fmt.Errorf("failed to start server: %w", err)
		}
		close(ch)
	}()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		close(app.MailQueue)
		func(SQL *sql.DB) {
			err := SQL.Close()
			if err != nil {

			}
		}(app.DB.SQL)

		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server gracefully: %w", err)
		}
		fmt.Println("Server shutdown gracefully")
		return nil
	}

}

// createDirIfNotExist creates a directory if it does not exist
func createDirIfNotExist(path string) error {
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, mode)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}
