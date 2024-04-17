package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/namhuydao/vigilate/internal/config"
	"github.com/namhuydao/vigilate/internal/handlers"
	"github.com/namhuydao/vigilate/internal/middleware"
	"log"
	"net/http"
)

func Routes() http.Handler {
	mux := chi.NewRouter()

	// default middleware
	mux.Use(middleware.SessionLoad)
	mux.Use(middleware.RecoverPanic)
	mux.Use(middleware.NoSurf)
	mux.Use(middleware.CheckRemember)
	mux.Use(middleware.NotFoundMiddleware)

	// login
	mux.Get("/", handlers.Repo.LoginScreen)
	mux.Post("/", handlers.Repo.Login)

	mux.Get("/user/logout", handlers.Repo.Logout)

	// our pusher routes
	mux.With(middleware.Auth).Route("/pusher", func(mux chi.Router) {
		mux.Post("/auth", handlers.Repo.PusherAuth)
	})

	// admin routes
	mux.With(middleware.Auth).Route("/admin", func(mux chi.Router) {
		// sample code for sending to private channel
		mux.Get("/private-message", handlers.Repo.SendPrivateMessage)

		// overview
		mux.Get("/overview", handlers.Repo.AdminOverview)
		mux.Get("/dashboard", handlers.Repo.AdminDashboard)

		// events
		mux.Get("/events", handlers.Repo.Events)

		// settings
		mux.Get("/settings", handlers.Repo.Settings)
		mux.Post("/settings", handlers.Repo.PostSettings)

		// service status pages (all hosts)
		mux.Get("/all-service-status/{status}", handlers.Repo.AllServices)

		// users
		mux.Get("/users", handlers.Repo.AllUsers)
		mux.Get("/user/{id}", handlers.Repo.OneUser)
		mux.Post("/user/{id}", handlers.Repo.PostOneUser)
		mux.Get("/user/delete/{id}", handlers.Repo.DeleteUser)

		// schedule
		mux.Get("/schedule", handlers.Repo.ListEntries)

		// preferences
		mux.Post("/preference/ajax/set-system-pref", handlers.Repo.SetSystemPref)
		mux.Post("/preference/ajax/toggle-monitoring", handlers.Repo.ToggleMonitoring)

		// hosts
		mux.Get("/host/all", handlers.Repo.AllHosts)
		mux.Get("/host/{id}", handlers.Repo.Host)
		mux.Post("/host/{id}", handlers.Repo.PostHost)
		mux.Post("/host/toggle-service", handlers.Repo.ToggleServiceForHost)
		mux.Get("/perform-check/{id}/{oldStatus}", handlers.Repo.TestCheck)
	})

	// static files
	fileServer := http.FileServer(http.Dir("./static/"))
	mux.With(middleware.FileMiddleware).Handle("/static/*", http.StripPrefix("/static", fileServer))

	populateKnownRoutes(mux)

	return mux
}

func populateKnownRoutes(router *chi.Mux) {
	walker := func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		config.KnownRoutes[route] = true
		return nil
	}

	if err := chi.Walk(router, walker); err != nil {
		log.Println(err)
	}
}
