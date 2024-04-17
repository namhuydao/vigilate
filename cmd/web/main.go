package main

import (
	"encoding/gob"
	"fmt"
	"github.com/namhuydao/vigilate/internal/setup"
	"log"
	"runtime"

	"github.com/namhuydao/vigilate/internal/config"
	"github.com/namhuydao/vigilate/internal/handlers"
	"github.com/namhuydao/vigilate/internal/models"

	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"
	"github.com/pusher/pusher-http-go"
)

var app config.AppConfig
var repo *handlers.DBRepo
var session *scs.SessionManager
var preferenceMap map[string]string
var wsClient pusher.Client

const vigilateVersion = "1.0.0"
const maxWorkerPoolSize = 5
const maxJobMaxWorkers = 5

func init() {
	gob.Register(models.User{})
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

// main is the application entry point
func main() {
	// set up application
	setup.NewSetUp(app, repo, session, preferenceMap, wsClient, vigilateVersion, maxWorkerPoolSize, maxJobMaxWorkers)
	insecurePort, err := setup.InitApp()
	if err != nil {
		log.Fatal(err)
	}

	// print info
	log.Printf("******************************************")
	log.Printf("** %sVigilate%s v%s built in %s", "\033[31m", "\033[0m", vigilateVersion, runtime.Version())
	log.Printf("**----------------------------------------")
	log.Printf("** Running with %d Processors", runtime.NumCPU())
	log.Printf("** Running on %s", runtime.GOOS)
	log.Printf("******************************************")

	err = setup.Start(insecurePort)
	if err != nil {
		fmt.Println("failed to start setup:", err)
	}
}
