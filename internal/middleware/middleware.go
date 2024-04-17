package middleware

import (
	"fmt"
	"github.com/namhuydao/vigilate/internal/config"
	"github.com/namhuydao/vigilate/internal/driver"
	"github.com/namhuydao/vigilate/internal/handlers"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/namhuydao/vigilate/internal/helpers"

	"github.com/justinas/nosurf"
)

var app *config.AppConfig
var repo *handlers.DBRepo

func NewMiddleware(a *config.AppConfig, db *driver.DB) {
	app = a
	repo = handlers.NewDBHandlers(db, a)
}

// SessionLoad loads the session on requests
func SessionLoad(next http.Handler) http.Handler {
	return app.Session.LoadAndSave(next)
}

func NotFoundMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		matched := false
		for known, _ := range config.KnownRoutes {
			if helpers.MatchRoutePath(r.URL.Path, known) {
				matched = true
				break
			}
		}

		if !matched {
			handlers.ClientError(w, r, http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func FileMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		referer := r.Header.Get("Referer")
		appDomain := fmt.Sprintf("http://%s:%s", os.Getenv("APP_DOMAIN"), os.Getenv("APP_PORT"))
		if referer == "" || !strings.HasPrefix(referer, appDomain) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Auth checks for authentication
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !helpers.IsAuthenticated(r) {
			url := r.URL.Path
			http.Redirect(w, r, fmt.Sprintf("/?target=%s", url), http.StatusFound)
			return
		}
		w.Header().Add("Cache-Control", "no-store")

		next.ServeHTTP(w, r)
	})
}

// RecoverPanic recovers from a panic
func RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			// Check if there has been a panic
			if err := recover(); err != nil {
				// return a 500 Internal Server response
				helpers.ServerError(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// NoSurf implements CSRF protection
func NoSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)

	csrfHandler.ExemptPath("/pusher/auth")
	csrfHandler.ExemptPath("/pusher/hook")

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   app.InProduction,
		SameSite: http.SameSiteStrictMode,
		Domain:   app.Domain,
	})

	return csrfHandler
}

// CheckRemember checks to see if we should log the user in automatically
func CheckRemember(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !helpers.IsAuthenticated(r) {
			cookie, err := r.Cookie(fmt.Sprintf("_%s_gowatcher_remember", app.PreferenceMap["identifier"]))
			if err != nil {
				next.ServeHTTP(w, r)
			} else {
				key := cookie.Value
				// have a remember token, so try to log the user in
				if len(key) > 0 {
					// key length > 0, so it might be a valid token
					split := strings.Split(key, "|")
					uid, hash := split[0], split[1]
					id, _ := strconv.Atoi(uid)
					validHash := repo.DB.CheckForRememberMeToken(id, hash)
					if validHash {
						// valid remember me token, so log the user in
						_ = app.Session.RenewToken(r.Context())
						user, _ := repo.DB.GetUserById(id)
						app.Session.Put(r.Context(), "userID", id)
						app.Session.Put(r.Context(), "userName", user.FirstName)
						app.Session.Put(r.Context(), "userFirstName", user.FirstName)
						app.Session.Put(r.Context(), "userLastName", user.LastName)
						app.Session.Put(r.Context(), "user", user)
						next.ServeHTTP(w, r)
					} else {
						// invalid token, so delete the cookie
						deleteRememberCookie(w, r)
						app.Session.Put(r.Context(), "error", "You've been logged out from another device!")
						next.ServeHTTP(w, r)
					}
				} else {
					// key length is zero, so it's a leftover cookie (user has not closed browser)
					next.ServeHTTP(w, r)
				}
			}
		} else {
			// they are logged in, but make sure that the remember token has not been revoked
			cookie, err := r.Cookie(fmt.Sprintf("_%s_gowatcher_remember", app.PreferenceMap["identifier"]))
			if err != nil {
				// no cookie
				next.ServeHTTP(w, r)
			} else {
				key := cookie.Value
				// have a remember token, but make sure it's valid
				if len(key) > 0 {
					split := strings.Split(key, "|")
					uid, hash := split[0], split[1]
					id, _ := strconv.Atoi(uid)
					validHash := repo.DB.CheckForRememberMeToken(id, hash)
					if !validHash {
						deleteRememberCookie(w, r)
						app.Session.Put(r.Context(), "error", "You've been logged out from another device!")
						next.ServeHTTP(w, r)
					} else {
						next.ServeHTTP(w, r)
					}
				} else {
					next.ServeHTTP(w, r)
				}
			}
		}
	})
}

// deleteRememberCookie deletes the remember me cookie, and logs the user out
func deleteRememberCookie(w http.ResponseWriter, r *http.Request) {
	_ = app.Session.RenewToken(r.Context())
	// delete the cookie
	newCookie := http.Cookie{
		Name:     fmt.Sprintf("_%s_gowatcher_remember", app.PreferenceMap["identifier"]),
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-100 * time.Hour),
		HttpOnly: true,
		Domain:   app.Domain,
		MaxAge:   -1,
		Secure:   app.InProduction,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, &newCookie)

	// log them out
	app.Session.Remove(r.Context(), "userID")
	_ = app.Session.Destroy(r.Context())
	_ = app.Session.RenewToken(r.Context())
}
