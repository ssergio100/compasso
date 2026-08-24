// Package localapi exposes the small local agent API on the system D-Bus.
package localapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/ssergio100/compasso/agent/localauth"
)

const (
	BusName       = "br.com.tempo.Agent"
	ObjectPath    = dbus.ObjectPath("/br/com/tempo/Agent")
	InterfaceName = "br.com.tempo.Agent"
)

// bonusAPI translates D-Bus requests into the local bonus service.
type bonusAPI struct {
	service         *localauth.Service
	synchronization synchronizationSource
	now             func() time.Time
}

type synchronizationSource interface {
	SynchronizationStatus() (checked, online bool)
}

type synchronizationReportSource interface {
	SynchronizationReport() (checked, online bool, detail string)
}

func newBonusAPI(service *localauth.Service, synchronization synchronizationSource) (*bonusAPI, error) {
	if service == nil {
		return nil, errors.New("local bonus service is required")
	}
	if synchronization == nil {
		return nil, errors.New("synchronization source is required")
	}
	return &bonusAPI{service: service, synchronization: synchronization, now: time.Now}, nil
}

// GetSynchronizationStatus returns the live heartbeat state without exposing
// credentials or connection error details.
func (a *bonusAPI) GetSynchronizationStatus() (string, *dbus.Error) {
	checked, online := a.synchronization.SynchronizationStatus()
	if !checked {
		return "checking", nil
	}
	if online {
		return "online", nil
	}
	return "offline", nil
}

// GetSynchronizationReport is the human-facing, additive successor to
// GetSynchronizationStatus. The old method remains available to clients from
// older packages, while new interfaces receive an actionable explanation.
func (a *bonusAPI) GetSynchronizationReport() (string, string, *dbus.Error) {
	checked, online := a.synchronization.SynchronizationStatus()
	detail := ""
	if source, ok := a.synchronization.(synchronizationReportSource); ok {
		checked, online, detail = source.SynchronizationReport()
	}
	if !checked {
		return "checking", "Aguardando a primeira resposta do servidor.", nil
	}
	if online {
		return "online", "", nil
	}
	return "offline", humanSynchronizationDetail(detail), nil
}

func humanSynchronizationDetail(detail string) string {
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "http 400"):
		return "O servidor recusou a sincronização (erro 400). O agente e o servidor podem estar em versões incompatíveis."
	case strings.Contains(lower, "http 401"), strings.Contains(lower, "http 403"):
		return "O servidor recusou a identificação deste computador. Revise o dispositivo e o token nas configurações."
	case strings.Contains(lower, "local revision"), strings.Contains(lower, "http 409"):
		return "O estado deste computador não corresponde ao cadastro atual do servidor. Revise a configuração do dispositivo."
	case strings.Contains(lower, "update the compasso agent"), strings.Contains(lower, "http 426"):
		return "Este agente precisa ser atualizado para continuar a comunicação com o servidor."
	case strings.Contains(lower, "http 502"), strings.Contains(lower, "http 503"), strings.Contains(lower, "http 504"):
		return "O servidor está temporariamente indisponível. O agente continuará tentando automaticamente."
	case strings.Contains(lower, "lookup "), strings.Contains(lower, "no such host"), strings.Contains(lower, "server misbehaving"):
		return "Não foi possível localizar o servidor na rede. Verifique a conexão e o endereço configurado."
	case strings.Contains(lower, "deadline exceeded"), strings.Contains(lower, "timeout"):
		return "O servidor demorou demais para responder. O agente continuará tentando automaticamente."
	default:
		return "A sincronização falhou. Abra as configurações do Compasso para revisar a comunicação."
	}
}

// AddLocalBonus is called by the unprivileged GTK dialog. The password is
// checked by the root agent and is never written to disk or logs.
func (a *bonusAPI) AddLocalBonus(password string, seconds uint32) (string, *dbus.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := a.service.Grant(ctx, password, int64(seconds), a.now())
	if err != nil {
		switch {
		case errors.Is(err, localauth.ErrPasswordNotConfigured):
			return "", dbus.NewError(BusName+".Error.PasswordNotConfigured", []interface{}{"Nenhuma senha de administrador foi cadastrada."})
		case errors.Is(err, localauth.ErrInvalidPassword):
			return "", dbus.NewError(BusName+".Error.InvalidPassword", []interface{}{"Senha incorreta."})
		case errors.Is(err, localauth.ErrRateLimited):
			return "", dbus.NewError(BusName+".Error.RateLimited", []interface{}{"Aguarde antes de tentar novamente."})
		default:
			return "", dbus.NewError(BusName+".Error.Failed", []interface{}{"Não foi possível adicionar o tempo."})
		}
	}
	return result.UUID, nil
}

// Server owns the system-bus connection and exported object.
type Server struct {
	connection *dbus.Conn
}

func ExportSystem(service *localauth.Service, synchronization synchronizationSource) (*Server, error) {
	api, err := newBonusAPI(service, synchronization)
	if err != nil {
		return nil, err
	}
	connection, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect system D-Bus: %w", err)
	}
	cleanup := func() { _ = connection.Close() }
	if err := connection.Export(api, ObjectPath, InterfaceName); err != nil {
		cleanup()
		return nil, fmt.Errorf("export local bonus API: %w", err)
	}
	node := &introspect.Node{
		Name: string(ObjectPath),
		Interfaces: []introspect.Interface{{
			Name: InterfaceName,
			Methods: []introspect.Method{{
				Name: "AddLocalBonus",
				Args: []introspect.Arg{
					{Name: "password", Type: "s", Direction: "in"},
					{Name: "seconds", Type: "u", Direction: "in"},
					{Name: "event_uuid", Type: "s", Direction: "out"},
				},
			}, {
				Name: "GetSynchronizationStatus",
				Args: []introspect.Arg{
					{Name: "status", Type: "s", Direction: "out"},
				},
			}, {
				Name: "GetSynchronizationReport",
				Args: []introspect.Arg{
					{Name: "status", Type: "s", Direction: "out"},
					{Name: "detail", Type: "s", Direction: "out"},
				},
			}},
		}, introspect.IntrospectData},
	}
	if err := connection.Export(introspect.NewIntrospectable(node), ObjectPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		cleanup()
		return nil, fmt.Errorf("export local API introspection: %w", err)
	}
	reply, err := connection.RequestName(BusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("request local agent D-Bus name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		cleanup()
		return nil, fmt.Errorf("D-Bus name %s is already owned", BusName)
	}
	return &Server{connection: connection}, nil
}

func (s *Server) Close() error {
	if s == nil || s.connection == nil {
		return nil
	}
	return s.connection.Close()
}
