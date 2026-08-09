// Package sessionlogout discovers an orderly logout capability on the current
// user's session bus. It never terminates processes or talks to logind.
package sessionlogout

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

var errNoSupportedProvider = errors.New("no supported session logout provider is available")

type logoutProvider struct {
	Name       string
	BusName    string
	ObjectPath dbus.ObjectPath
	Method     string
	Arguments  []interface{}
}

var orderlyLogoutProviders = []logoutProvider{
	{
		Name: "plasma-session", BusName: "org.kde.Shutdown",
		ObjectPath: "/Shutdown", Method: "org.kde.Shutdown.logout",
	},
	{
		Name: "gnome-session", BusName: "org.gnome.SessionManager",
		ObjectPath: "/org/gnome/SessionManager", Method: "org.gnome.SessionManager.Logout",
		Arguments: []interface{}{uint32(1)},
	},
}

type sessionBus interface {
	NameAvailable(context.Context, string) (bool, error)
	Call(context.Context, logoutProvider) error
}

type dbusSessionBus struct {
	connection *dbus.Conn
}

// Connect opens the calling user's session bus. The caller owns the returned
// connection and must close it.
func Connect() (*dbus.Conn, error) {
	connection, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to user session bus: %w", err)
	}
	return connection, nil
}

// Request asks an available session manager to perform its normal logout.
// Unsupported sessions fail safely and remain open.
func Request(ctx context.Context, connection *dbus.Conn) (string, error) {
	if connection == nil {
		return "", errors.New("session bus connection is required")
	}
	return request(ctx, dbusSessionBus{connection: connection})
}

func request(ctx context.Context, bus sessionBus) (string, error) {
	var failures []string
	for _, provider := range orderlyLogoutProviders {
		available, err := bus.NameAvailable(ctx, provider.BusName)
		if err != nil {
			failures = append(failures, provider.Name+" discovery: "+err.Error())
			continue
		}
		if !available {
			continue
		}
		if err := bus.Call(ctx, provider); err != nil {
			failures = append(failures, provider.Name+" logout: "+err.Error())
			continue
		}
		return provider.Name, nil
	}
	if len(failures) != 0 {
		return "", fmt.Errorf("%w: %s", errNoSupportedProvider, strings.Join(failures, "; "))
	}
	return "", errNoSupportedProvider
}

func (b dbusSessionBus) NameAvailable(ctx context.Context, name string) (bool, error) {
	var hasOwner bool
	call := b.connection.BusObject().CallWithContext(
		ctx, "org.freedesktop.DBus.NameHasOwner", 0, name,
	)
	if call.Err != nil {
		return false, call.Err
	}
	if err := call.Store(&hasOwner); err != nil {
		return false, err
	}
	if hasOwner {
		return true, nil
	}
	var activatableNames []string
	call = b.connection.BusObject().CallWithContext(
		ctx, "org.freedesktop.DBus.ListActivatableNames", 0,
	)
	if call.Err != nil {
		return false, call.Err
	}
	if err := call.Store(&activatableNames); err != nil {
		return false, err
	}
	return registeredNameAvailable(false, activatableNames, name), nil
}

func registeredNameAvailable(hasOwner bool, activatableNames []string, name string) bool {
	if hasOwner {
		return true
	}
	for _, activatableName := range activatableNames {
		if activatableName == name {
			return true
		}
	}
	return false
}

func (b dbusSessionBus) Call(ctx context.Context, provider logoutProvider) error {
	return b.connection.Object(provider.BusName, provider.ObjectPath).
		CallWithContext(ctx, provider.Method, 0, provider.Arguments...).Err
}
