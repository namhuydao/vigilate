package helpers

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/justinas/nosurf"
	"github.com/namhuydao/vigilate/internal/config"
	"github.com/namhuydao/vigilate/internal/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var app *config.AppConfig
var src = rand.NewSource(time.Now().UnixNano())
var atExpires = time.Now().Add(time.Minute * 30).Unix()

// File extension for template files
const templateFileExt = ".gohtml"
const layoutPath = "./views/layouts/"
const templatePath = "./views/templates"
const partialPath = "./views/partials"

// TemplateData defines template data
type TemplateData struct {
	CSRFToken       string
	IsAuthenticated bool
	PreferenceMap   map[string]string
	User            models.User
	Flash           string
	Warning         string
	Error           string
	GwVersion       string
	DataMap         map[string]any
}

// TemplateRenderer represents a template renderer
type TemplateRenderer struct {
	templates *template.Template
}

type APIResponse struct {
	Data        interface{} `json:"data"`
	AccessToken string      `json:"access_token,omitempty"`
	Code        int         `json:"code"`
	PageIndex   int         `json:"page_index,omitempty"`
	PageSize    int         `json:"page_size,omitempty"`
	Total       int         `json:"total"`
	Status      string      `json:"status,omitempty"`
	Message     string      `json:"message"`
}

// NewHelpers creates new helpers
func NewHelpers(a *config.AppConfig) {
	app = a
}

// HumanDate formats a time in YYYY-MM-DD format
func HumanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// FormatDateWithLayout formats a time with provided (go compliant) format string, and returns it as a string
func FormatDateWithLayout(t time.Time, f string) string {
	return t.Format(f)
}

// DateAfterY1 is used to verify that a date is after the year 1 (since go hates nulls)
func DateAfterY1(t time.Time) bool {
	yearOne := time.Date(0001, 11, 17, 20, 34, 58, 651387237, time.UTC)
	return t.After(yearOne)
}

// AddDefaultData adds default data which is accessible to all templates
func AddDefaultData(td TemplateData, r *http.Request) TemplateData {
	td.CSRFToken = nosurf.Token(r)
	td.IsAuthenticated = IsAuthenticated(r)
	td.PreferenceMap = app.PreferenceMap
	// if logged in, store user id in template data
	if td.IsAuthenticated {
		u := app.Session.Get(r.Context(), "user").(models.User)
		td.User = u
	}

	td.Flash = app.Session.PopString(r.Context(), "flash")
	td.Warning = app.Session.PopString(r.Context(), "warning")
	td.Error = app.Session.PopString(r.Context(), "error")

	return td
}

func (tr *TemplateRenderer) RenderTemplate(w http.ResponseWriter, r *http.Request, template string, data TemplateData) error {
	data = AddDefaultData(data, r)

	// Execute the specific template with data
	err := tr.templates.ExecuteTemplate(w, template, data)
	if err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}

	return nil
}

func NewTemplateRenderer(templatePath, templateName string) (*TemplateRenderer, error) {
	// Parse template files
	templateFile, err := filepath.Glob(filepath.Join(templatePath, fmt.Sprintf("%s%s", templateName, templateFileExt)))
	if err != nil {
		return nil, fmt.Errorf("error finding template files: %v", err)
	}

	// Parse partials templates
	partialFiles, err := filepath.Glob(filepath.Join(partialPath, fmt.Sprintf("*%s", templateFileExt)))
	if err != nil {
		return nil, fmt.Errorf("error finding partials template files: %v", err)
	}

	allFiles := append(partialFiles, templateFile...)

	// Parse template
	templateGo, err := template.New("").Funcs(template.FuncMap{
		"humanDate":        HumanDate,
		"dateFromLayout":   FormatDateWithLayout,
		"dateAfterYearOne": DateAfterY1,
	}).ParseFiles(allFiles...)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %v", err)
	}

	return &TemplateRenderer{
		templates: templateGo,
	}, nil
}

func TemplateRender(w http.ResponseWriter, r *http.Request, templatePath string, templateName string, data TemplateData) error {
	// Create a new template renderer
	renderer, err := NewTemplateRenderer(templatePath, templateName)
	if err != nil {
		return err
	}
	return renderer.RenderTemplate(w, r, fmt.Sprintf("%s%s", templateName, templateFileExt), data)
}

func HxRender(w http.ResponseWriter, r *http.Request, templateName string, data TemplateData, printTemplateError func(w http.ResponseWriter, err error)) {
	var path string
	var name string
	if r.Header.Get("HX-Request") == "true" {
		path = templatePath
		name = templateName

	} else {
		path = layoutPath
		name = "layout"
	}

	err := TemplateRender(w, r, path, name, data)
	if err != nil {
		printTemplateError(w, err)
	}
}

// SendEmail sends an email
func SendEmail(mailMessage config.MailData) {
	if mailMessage.FromAddress == "" {
		mailMessage.FromAddress = app.PreferenceMap["smtp_from_email"]
		mailMessage.FromName = app.PreferenceMap["smtp_from_name"]
	}

	job := config.MailJob{MailMessage: mailMessage}
	app.MailQueue <- job
}

func IsAuthenticated(r *http.Request) bool {
	return app.Session.Get(r.Context(), "user") != nil
}

func CapitalizedString(str string) string {
	return cases.Title(language.English).String(str)
}

// MatchRoutePath check if the requested path matches the route path
func MatchRoutePath(path, routePath string) bool {
	pathSegments := strings.Split(path, "/")
	routeSegments := strings.Split(routePath, "/")

	if len(pathSegments) != len(routeSegments) {
		return false
	}

	for i, segment := range routeSegments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			continue
		}
		if pathSegments[i] != segment {
			return false
		}
	}

	return true
}

// RandomString returns a random string of letters of length n
func RandomString(n int) string {
	b := make([]byte, n)

	for i, theCache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			theCache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(theCache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		theCache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// ServerError will display error page for internal server error
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	_ = log.Output(2, trace)

	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Connection", "close")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	http.ServeFile(w, r, "./ui/static/500.html")
}

func GenerateAccessToken(userId uint) (string, int64, error) {
	accessTokenClaims := jwt.MapClaims{}
	accessTokenClaims["authorized"] = true
	accessTokenClaims["user_id"] = userId
	accessTokenClaims["exp"] = atExpires

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)

	tokenString, err := accessToken.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		return "", 0, err
	}

	return tokenString, atExpires, nil
}

func WriteResponse(w http.ResponseWriter, statusCode int, msg string) {
	response := APIResponse{
		Code:    statusCode,
		Message: msg,
	}

	jsonResponse, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err := w.Write(jsonResponse)
	if err != nil {
		return
	}
}

func WriteResponseWithModel(w http.ResponseWriter, model interface{}, statusCode int, msg string) {
	response := APIResponse{
		Data:    model,
		Code:    statusCode,
		Message: msg,
	}

	jsonResponse, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err := w.Write(jsonResponse)
	if err != nil {
		return
	}
}

func WriteResponseWithPagination(w http.ResponseWriter, model interface{}, pageIndex int, pageSize int, total, statusCode int, msg string) {
	response := APIResponse{
		Data:      model,
		PageIndex: pageIndex,
		PageSize:  pageSize,
		Total:     total,
		Code:      statusCode,
		Message:   msg,
	}

	jsonResponse, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err := w.Write(jsonResponse)
	if err != nil {
		return
	}
}

func WriteResponseWithToken(w http.ResponseWriter, model interface{}, token string, statusCode int, msg string) {
	response := APIResponse{
		Data:        model,
		Code:        statusCode,
		AccessToken: token,
		Message:     msg,
	}

	jsonResponse, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err := w.Write(jsonResponse)
	if err != nil {
		return
	}
}

//func CheckTokenHealth(accessToken models.AccessToken) (bool, error) {
//	if accessToken.AtExpires < time.Now().Unix() {
//		return false, models.ErrorTokenExpired
//	}
//
//	return true, nil
//}
//
//func extractTokenFromHeader(r *http.Request) string {
//	bearerToken := r.Header.Get("access_token")
//	if bearerToken == "" {
//		return ""
//	}
//
//	tokenParts := strings.Split(bearerToken, " ")
//	if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
//		return ""
//	}
//
//	return tokenParts[1]
//}
