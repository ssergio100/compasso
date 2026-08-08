package alert

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const desktopNotificationTimeout = 5 * time.Second

type commandExecutor func(context.Context, string, ...string) ([]byte, error)

// DesktopNotifier asks the controlled user's systemd manager to display a
// freedesktop notification inside the graphical session.
type DesktopNotifier struct {
	controlledUser string
	systemdRunPath string
	notifySendPath string
	executeCommand commandExecutor
}

func NewDesktopNotifier(controlledUser string) (*DesktopNotifier, error) {
	if controlledUser == "" || strings.ContainsAny(controlledUser, " \t\r\n/@") {
		return nil, errors.New("controlled user is invalid for desktop notifications")
	}
	return &DesktopNotifier{
		controlledUser: controlledUser,
		systemdRunPath: "/usr/bin/systemd-run",
		notifySendPath: "/usr/bin/notify-send",
		executeCommand: executeDesktopCommand,
	}, nil
}

func (notifier *DesktopNotifier) Notify(ctx context.Context, scheduledAlert Alert) error {
	if notifier == nil || scheduledAlert.Title == "" || scheduledAlert.Body == "" {
		return errors.New("desktop notifier and alert contents are required")
	}
	notificationContext, cancelNotification := context.WithTimeout(ctx, desktopNotificationTimeout)
	defer cancelNotification()
	machineName := notifier.controlledUser + "@.host"
	output, err := notifier.executeCommand(notificationContext, notifier.systemdRunPath,
		"--user", "--machine="+machineName, "--collect", "--quiet",
		notifier.notifySendPath, "--app-name=Compasso", "--urgency=critical",
		"--hint=string:sound-name:alarm-clock-elapsed",
		scheduledAlert.Title, scheduledAlert.Body,
	)
	if err != nil {
		return fmt.Errorf("display desktop alert: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func executeDesktopCommand(ctx context.Context, commandPath string, arguments ...string) ([]byte, error) {
	return exec.CommandContext(ctx, commandPath, arguments...).CombinedOutput()
}
