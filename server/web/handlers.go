package web

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sergio/compasso/agent/localauth"
	"github.com/sergio/compasso/agent/policy"
	"github.com/sergio/compasso/server/storage"
)

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if _, _, ok := a.authenticated(r); ok {
			http.Redirect(w, r, "/devices", http.StatusSeeOther)
			return
		}
		token, err := randomToken()
		if err != nil {
			http.Error(w, "Falha interna.", http.StatusInternalServerError)
			return
		}
		a.setCookie(w, &http.Cookie{Name: loginCSRFCookie, Value: token, HttpOnly: true, MaxAge: 600})
		a.render(w, "login", http.StatusOK, pageData{Title: "Entrar", CSRF: token})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Método não permitido.", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido.", http.StatusBadRequest)
		return
	}
	csrfCookie, err := r.Cookie(loginCSRFCookie)
	if err != nil || !constantEqual(csrfCookie.Value, r.FormValue("csrf_token")) {
		http.Error(w, "Token CSRF inválido.", http.StatusForbidden)
		return
	}
	admin, err := a.store.AdminByLogin(r.Context(), r.FormValue("login"))
	valid := false
	if err == nil && admin.Active {
		valid, _ = localauth.VerifyPassword(r.FormValue("password"), admin.PasswordHash)
	}
	if !valid {
		a.render(w, "login", http.StatusUnauthorized, pageData{
			Title: "Entrar", CSRF: csrfCookie.Value, Error: "Usuário ou senha inválidos.",
		})
		return
	}
	token, value, err := a.sessions.create(admin.ID, admin.Login, a.now())
	if err != nil {
		http.Error(w, "Falha interna.", http.StatusInternalServerError)
		return
	}
	a.setCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, HttpOnly: true,
		Expires: value.Expires, MaxAge: int(a.sessions.lifetime.Seconds()),
	})
	a.setCookie(w, &http.Cookie{Name: loginCSRFCookie, MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, "/devices", http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido.", http.StatusMethodNotAllowed)
		return
	}
	current, token, ok := a.requireSession(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil || !constantEqual(r.FormValue("csrf_token"), current.CSRF) {
		http.Error(w, "Token CSRF inválido.", http.StatusForbidden)
		return
	}
	a.sessions.delete(token)
	a.setCookie(w, &http.Cookie{Name: sessionCookieName, MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) devices(w http.ResponseWriter, r *http.Request) {
	current, _, ok := a.requireSession(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulário inválido.", http.StatusBadRequest)
			return
		}
		if !constantEqual(r.FormValue("csrf_token"), current.CSRF) {
			http.Error(w, "Token CSRF inválido.", http.StatusForbidden)
			return
		}
		device, err := a.store.CreateDevice(r.Context(), r.FormValue("name"), a.now())
		if err != nil {
			a.renderDevices(w, r, current, http.StatusBadRequest, err.Error())
			return
		}
		http.Redirect(w, r, "/devices/"+device.ID+"?success="+url.QueryEscape("Dispositivo criado."), http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido.", http.StatusMethodNotAllowed)
		return
	}
	a.renderDevices(w, r, current, http.StatusOK, "")
}

func (a *App) renderDevices(w http.ResponseWriter, r *http.Request, current session, status int, message string) {
	devices, err := a.store.ListDevices(r.Context())
	if err != nil {
		http.Error(w, "Falha ao carregar dispositivos.", http.StatusInternalServerError)
		return
	}
	for index := range devices {
		devices[index].Online = isOnline(devices[index].LastSeenAt, a.now(), a.onlineTimeout)
	}
	a.render(w, "devices", status, pageData{
		Title: "Dispositivos", Login: current.Login, CSRF: current.CSRF,
		Devices: devices, Error: message, Success: r.URL.Query().Get("success"),
	})
}

func (a *App) deviceRoutes(w http.ResponseWriter, r *http.Request) {
	current, _, ok := a.requireSession(w, r)
	if !ok {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/devices/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" || len(parts) > 3 {
		http.NotFound(w, r)
		return
	}
	deviceID := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		a.renderDevice(w, r, current, deviceID, http.StatusOK, "", "")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido.", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido.", http.StatusBadRequest)
		return
	}
	if !constantEqual(r.FormValue("csrf_token"), current.CSRF) {
		http.Error(w, "Token CSRF inválido.", http.StatusForbidden)
		return
	}
	a.handleDevicePost(w, r, current, deviceID, parts[1:])
}

func (a *App) handleDevicePost(w http.ResponseWriter, r *http.Request, current session, deviceID string, action []string) {
	var err error
	success := "Alterações salvas."
	switch {
	case len(action) == 1 && action[0] == "rename":
		err = a.store.RenameDevice(r.Context(), deviceID, r.FormValue("name"), a.now())
		success = "Nome atualizado."
	case len(action) == 1 && action[0] == "delete":
		err = a.store.DeleteDevice(r.Context(), deviceID)
		if err == nil {
			http.Redirect(w, r, "/devices?success="+url.QueryEscape("Dispositivo excluído."), http.StatusSeeOther)
			return
		}
	case len(action) == 1 && action[0] == "quotas":
		var quotas [7]int64
		for day := range quotas {
			quotas[day], err = parseDuration(r.FormValue(fmt.Sprintf("quota_%d", day)))
			if err != nil {
				break
			}
		}
		warning := 10
		if err == nil {
			warning, err = strconv.Atoi(r.FormValue("warning_minutes"))
		}
		if err == nil {
			err = a.store.SaveQuotas(r.Context(), deviceID, quotas, warning, a.now())
		}
		success = "Cotas atualizadas; aguardando sincronização."
	case len(action) == 1 && action[0] == "routines":
		routine := storage.Routine{ID: r.FormValue("routine_id"), Name: r.FormValue("name"), Enabled: true}
		for day := range routine.Days {
			routine.Days[day] = r.FormValue(fmt.Sprintf("day_%d", day)) == "on"
		}
		routine.Start, err = parseClock(r.FormValue("start"))
		if err == nil {
			routine.End, err = parseClock(r.FormValue("end"))
		}
		if err == nil {
			_, err = a.store.SaveRoutine(r.Context(), deviceID, routine, a.now())
		}
		success = "Rotina salva; aguardando sincronização."
	case len(action) == 2 && action[0] == "routines" && action[1] != "":
		err = a.store.DeleteRoutine(r.Context(), deviceID, action[1], a.now())
		success = "Rotina excluída."
	case len(action) == 1 && action[0] == "password":
		password, confirmation := r.FormValue("password"), r.FormValue("password_confirmation")
		if password == "" || password != confirmation {
			err = errors.New("a senha e a confirmação precisam ser iguais")
		} else {
			var verifier string
			verifier, err = localauth.HashPassword(password, localauth.DefaultArgon2Params)
			if err == nil {
				err = a.store.SetLocalPassword(r.Context(), deviceID, verifier, a.now())
			}
		}
		success = "Senha local alterada; aguardando sincronização."
	case len(action) == 1 && action[0] == "token":
		var token string
		token, err = a.store.IssueDeviceToken(r.Context(), deviceID, a.now())
		if err == nil {
			a.renderDevice(w, r, current, deviceID, http.StatusOK, "", token)
			return
		}
	case len(action) == 1 && action[0] == "revoke-token":
		err = a.store.RevokeDeviceToken(r.Context(), deviceID, a.now())
		success = "Token revogado. O agente não poderá sincronizar até receber uma nova credencial."
	case len(action) == 1 && action[0] == "control":
		kind := r.FormValue("command")
		err = a.store.QueueControl(r.Context(), deviceID, kind, a.now())
		success = "Comando enviado; será aplicado no próximo heartbeat."
	case len(action) == 1 && action[0] == "bonus":
		var minutes int
		minutes, err = strconv.Atoi(r.FormValue("minutes"))
		if err == nil && (minutes <= 0 || minutes > 12*60) {
			err = errors.New("o bônus deve ser de 1 minuto a 12 horas")
		}
		if err == nil {
			err = a.store.QueueRemoteBonus(r.Context(), deviceID, a.now().Format("2006-01-02"), int64(minutes*60), a.now())
		}
		success = "Tempo extra enviado; será aplicado no próximo heartbeat."
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.renderDevice(w, r, current, deviceID, http.StatusBadRequest, err.Error(), "")
		return
	}
	http.Redirect(w, r, "/devices/"+deviceID+"?success="+url.QueryEscape(success), http.StatusSeeOther)
}

func (a *App) renderDevice(w http.ResponseWriter, r *http.Request, current session, deviceID string, status int, message, deviceToken string) {
	device, storedPolicy, err := a.store.LoadDevice(r.Context(), deviceID)
	if errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Falha ao carregar dispositivo.", http.StatusInternalServerError)
		return
	}
	summary, err := a.store.LoadDailySummary(r.Context(), deviceID, a.now().Format("2006-01-02"))
	if err != nil {
		http.Error(w, "Falha ao carregar uso diário.", http.StatusInternalServerError)
		return
	}
	events, err := a.store.ListAudit(r.Context(), deviceID, 30)
	if err != nil {
		http.Error(w, "Falha ao carregar histórico.", http.StatusInternalServerError)
		return
	}
	todayQuota := storedPolicy.WeeklyQuota[a.now().Weekday()]
	remaining := todayQuota + summary.BonusSeconds - summary.UsedSeconds
	if remaining < 0 {
		remaining = 0
	}
	nextBlock := "Aguardando sincronização"
	decisionInput := policy.Input{
		Now: a.now(), Quota: secondsQuota(storedPolicy.WeeklyQuota),
		ManualBlock: storedPolicy.ManualBlock,
		Consumed:    time.Duration(summary.UsedSeconds) * time.Second,
		Bonus:       time.Duration(summary.BonusSeconds) * time.Second,
	}
	if storedPolicy.MonitoringPaused {
		decisionInput.Monitoring = policy.MonitoringPaused
	}
	for _, routine := range storedPolicy.Routines {
		if !routine.Enabled {
			continue
		}
		decisionInput.Routines = append(decisionInput.Routines, policy.Routine{
			Name: routine.Name, Days: routine.Days,
			Start: time.Duration(routine.Start) * time.Second, End: time.Duration(routine.End) * time.Second,
		})
	}
	online := isOnline(device.LastSeenAt, a.now(), a.onlineTimeout)
	counting := false
	if decision, evaluationErr := policy.Evaluate(decisionInput); evaluationErr == nil {
		counting = decision.ShouldCount && online
		if !decision.NextBlockAt.IsZero() {
			nextBlock = decision.NextBlockAt.Format("02/01 15:04")
		}
	}
	var editRoutine *storage.Routine
	if editID := r.URL.Query().Get("edit_routine"); editID != "" {
		for index := range storedPolicy.Routines {
			if storedPolicy.Routines[index].ID == editID {
				copy := storedPolicy.Routines[index]
				editRoutine = &copy
			}
		}
	}
	a.render(w, "device", status, pageData{
		Title: device.Name, Login: current.Login, CSRF: current.CSRF,
		Device: device, Policy: storedPolicy, Events: events, EditRoutine: editRoutine,
		TodayQuota: todayQuota, TodayUsed: summary.UsedSeconds,
		Remaining: remaining, Counting: counting, NextBlock: nextBlock, PasswordSet: storedPolicy.LocalPasswordVerifier != "",
		Online: online, LastSeen: formatLastSeen(device.LastSeenAt),
		DeviceToken:  deviceToken,
		WeekdayNames: []string{"Domingo", "Segunda", "Terça", "Quarta", "Quinta", "Sexta", "Sábado"},
		Error:        message, Success: r.URL.Query().Get("success"),
	})
}

func isOnline(lastSeen *time.Time, now time.Time, timeout time.Duration) bool {
	return lastSeen != nil && !lastSeen.Before(now.Add(-timeout))
}

func formatLastSeen(lastSeen *time.Time) string {
	if lastSeen == nil {
		return "Nunca conectado"
	}
	return lastSeen.Local().Format("02/01/2006 15:04:05")
}

func parseDuration(value string) (int64, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, errors.New("use o formato HH:MM para as cotas")
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, errors.New("cota inválida")
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil || hours < 0 || hours > 24 || minutes < 0 || minutes > 59 || (hours == 24 && minutes != 0) {
		return 0, errors.New("cota inválida")
	}
	return int64(hours*3600 + minutes*60), nil
}

func parseClock(value string) (int64, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, errors.New("horário inválido")
	}
	return int64(parsed.Hour()*3600 + parsed.Minute()*60), nil
}

func secondsQuota(stored [7]int64) (converted policy.WeeklyQuota) {
	for day, seconds := range stored {
		converted[day] = time.Duration(seconds) * time.Second
	}
	return converted
}
