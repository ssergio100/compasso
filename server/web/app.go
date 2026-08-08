// Package web implements the server-rendered administrative panel.
package web

import (
	"crypto/subtle"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/sergio/compasso/server/storage"
)

const (
	sessionCookieName = "tempo_admin_session"
	loginCSRFCookie   = "tempo_login_csrf"
)

//go:embed templates/*.html static/*
var assets embed.FS

type App struct {
	store         *storage.Store
	sessions      *sessionStore
	secureCookies bool
	onlineTimeout time.Duration
	now           func() time.Time
	templates     map[string]*template.Template
	handler       http.Handler
}

type pageData struct {
	Title        string
	Login        string
	CSRF         string
	Error        string
	Success      string
	Devices      []storage.Device
	Device       storage.Device
	Policy       storage.Policy
	Events       []storage.AuditEvent
	EditRoutine  *storage.Routine
	TodayUsed    int64
	TodayBonus   int64
	Remaining    int64
	NextBlock    string
	PasswordSet  bool
	Online       bool
	LastSeen     string
	DeviceToken  string
	WeekdayNames []string
}

func New(store *storage.Store, secureCookies bool, sessionLifetime time.Duration, configuredOnlineTimeout ...time.Duration) (*App, error) {
	if store == nil {
		return nil, fmt.Errorf("server store is required")
	}
	functions := template.FuncMap{
		"duration": formatDuration,
		"clock":    formatClock,
		"dayNames": selectedDayNames,
		"daySelected": func(days [7]bool, day int) bool {
			return day >= 0 && day < len(days) && days[day]
		},
		"eventName": func(kind string) string {
			labels := map[string]string{
				"device_created": "Dispositivo criado", "device_renamed": "Dispositivo renomeado",
				"quotas_updated": "Cotas atualizadas", "routine_saved": "Rotina salva",
				"routine_deleted": "Rotina excluída", "local_password_changed": "Senha local alterada",
				"device_token_issued": "Credencial do agente gerada", "bonus_added": "Tempo extra adicionado",
				"pause_monitoring": "Vigilância pausada", "resume_monitoring": "Vigilância retomada",
				"block_now": "Bloqueio imediato", "clear_manual_block": "Bloqueio removido",
			}
			if label := labels[kind]; label != "" {
				return label
			}
			return kind
		},
	}
	templates := make(map[string]*template.Template)
	for _, page := range []string{"login", "devices", "device"} {
		parsed, err := template.New("base.html").Funcs(functions).ParseFS(assets, "templates/base.html", "templates/"+page+".html")
		if err != nil {
			return nil, fmt.Errorf("parse %s template: %w", page, err)
		}
		templates[page] = parsed
	}
	onlineTimeout := 60 * time.Second
	if len(configuredOnlineTimeout) != 0 {
		onlineTimeout = configuredOnlineTimeout[0]
	}
	if onlineTimeout <= 0 {
		return nil, fmt.Errorf("online timeout must be positive")
	}
	app := &App{
		store: store, sessions: newSessionStore(sessionLifetime), secureCookies: secureCookies,
		now: time.Now, templates: templates, onlineTimeout: onlineTimeout,
	}
	mux := http.NewServeMux()
	staticFS, _ := fs.Sub(assets, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/login", app.login)
	mux.HandleFunc("/logout", app.logout)
	mux.HandleFunc("/api/v1/device/heartbeat", app.heartbeat)
	mux.HandleFunc("/devices", app.devices)
	mux.HandleFunc("/devices/", app.deviceRoutes)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/devices", http.StatusSeeOther)
	})
	app.handler = securityHeaders(mux)
	return app, nil
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) { a.handler.ServeHTTP(w, r) }

func (a *App) render(w http.ResponseWriter, page string, status int, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := a.templates[page].ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Falha ao renderizar a página.", http.StatusInternalServerError)
	}
}

func (a *App) authenticated(r *http.Request) (session, string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return session{}, "", false
	}
	value, ok := a.sessions.get(cookie.Value, a.now())
	return value, cookie.Value, ok
}

func (a *App) requireSession(w http.ResponseWriter, r *http.Request) (session, string, bool) {
	value, token, ok := a.authenticated(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return session{}, "", false
	}
	return value, token, true
}

func (a *App) setCookie(w http.ResponseWriter, cookie *http.Cookie) {
	cookie.Path = "/"
	cookie.Secure = a.secureCookies
	cookie.SameSite = http.SameSiteStrictMode
	http.SetCookie(w, cookie)
}

func constantEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; img-src 'self' data:; form-action 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func formatDuration(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%02d:%02d", seconds/3600, (seconds%3600)/60)
}

func formatClock(seconds int64) string {
	return fmt.Sprintf("%02d:%02d", seconds/3600, (seconds%3600)/60)
}

func selectedDayNames(days [7]bool) string {
	names := []string{"Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"}
	var selected []string
	for index, enabled := range days {
		if enabled {
			selected = append(selected, names[index])
		}
	}
	return strings.Join(selected, ", ")
}
