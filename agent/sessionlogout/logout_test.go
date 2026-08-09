package sessionlogout

import (
	"context"
	"errors"
	"testing"
)

type fakeSessionBus struct {
	available map[string]bool
	calls     []logoutProvider
	fail      map[string]error
}

func (f *fakeSessionBus) NameAvailable(_ context.Context, name string) (bool, error) {
	return f.available[name], nil
}

func (f *fakeSessionBus) Call(_ context.Context, provider logoutProvider) error {
	f.calls = append(f.calls, provider)
	return f.fail[provider.Name]
}

func TestRegisteredActivatableProviderIsAvailableBeforeItHasOwner(t *testing.T) {
	if !registeredNameAvailable(false, []string{"org.kde.Shutdown"}, "org.kde.Shutdown") {
		t.Fatal("activatable Plasma logout provider was treated as unavailable")
	}
	if registeredNameAvailable(false, nil, "org.kde.Shutdown") {
		t.Fatal("unregistered provider was treated as available")
	}
}

func TestRequestUsesAvailableProviderWithExactArguments(t *testing.T) {
	bus := &fakeSessionBus{available: map[string]bool{"org.gnome.SessionManager": true}}
	provider, err := request(context.Background(), bus)
	if err != nil || provider != "gnome-session" || len(bus.calls) != 1 {
		t.Fatalf("provider=%q calls=%+v err=%v", provider, bus.calls, err)
	}
	if len(bus.calls[0].Arguments) != 1 || bus.calls[0].Arguments[0] != uint32(1) {
		t.Fatalf("GNOME logout arguments=%v", bus.calls[0].Arguments)
	}
}

func TestRequestTriesAnotherOwnedProviderAfterSafeFailure(t *testing.T) {
	bus := &fakeSessionBus{
		available: map[string]bool{
			"org.kde.Shutdown": true, "org.gnome.SessionManager": true,
		},
		fail: map[string]error{"plasma-session": errors.New("refused")},
	}
	provider, err := request(context.Background(), bus)
	if err != nil || provider != "gnome-session" || len(bus.calls) != 2 {
		t.Fatalf("provider=%q calls=%+v err=%v", provider, bus.calls, err)
	}
}

func TestUnknownSessionFailsWithoutTerminationFallback(t *testing.T) {
	bus := &fakeSessionBus{available: map[string]bool{}}
	if _, err := request(context.Background(), bus); !errors.Is(err, errNoSupportedProvider) {
		t.Fatalf("unknown session error=%v", err)
	}
	if len(bus.calls) != 0 {
		t.Fatalf("unknown session invoked providers: %+v", bus.calls)
	}
}
