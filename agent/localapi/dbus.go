// Package localapi exposes the small local agent API on the system D-Bus.
package localapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/sergio/compasso/agent/localauth"
)

const (
	BusName       = "br.com.tempo.Agent"
	ObjectPath    = dbus.ObjectPath("/br/com/tempo/Agent")
	InterfaceName = "br.com.tempo.Agent"
)

// BonusAPI translates D-Bus requests into the tested local bonus service.
type BonusAPI struct {
	service *localauth.Service
	now     func() time.Time
}

func NewBonusAPI(service *localauth.Service) (*BonusAPI, error) {
	if service == nil {
		return nil, errors.New("local bonus service is required")
	}
	return &BonusAPI{service: service, now: time.Now}, nil
}

// AddLocalBonus is called by the unprivileged GTK dialog. The password is
// checked by the root agent and is never written to disk or logs.
func (a *BonusAPI) AddLocalBonus(password string, seconds uint32) (string, *dbus.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := a.service.Grant(ctx, password, int64(seconds), a.now())
	if err != nil {
		switch {
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

func ExportSystem(service *localauth.Service) (*Server, error) {
	api, err := NewBonusAPI(service)
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
