package alert

import (
	"context"
	"reflect"
	"testing"
)

func TestDesktopNotifierTargetsControlledUserSession(t *testing.T) {
	notifier, err := NewDesktopNotifier("child")
	if err != nil {
		t.Fatal(err)
	}
	var commandPath string
	var commandArguments []string
	notifier.executeCommand = func(_ context.Context, path string, arguments ...string) ([]byte, error) {
		commandPath = path
		commandArguments = append([]string(nil), arguments...)
		return nil, nil
	}
	if err := notifier.Notify(context.Background(), Alert{Title: "Bloqueio em 1 minuto", Body: "Salve seu jogo."}); err != nil {
		t.Fatal(err)
	}
	expectedArguments := []string{
		"--user", "--machine=child@.host", "--collect", "--quiet",
		"/usr/bin/notify-send", "--app-name=Compasso", "--urgency=critical",
		"--hint=string:sound-name:alarm-clock-elapsed",
		"Bloqueio em 1 minuto", "Salve seu jogo.",
	}
	if commandPath != "/usr/bin/systemd-run" || !reflect.DeepEqual(commandArguments, expectedArguments) {
		t.Fatalf("command=%q arguments=%q", commandPath, commandArguments)
	}
}
