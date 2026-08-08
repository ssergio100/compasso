package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sergio/compasso/agent/config"
	"github.com/sergio/compasso/agent/pamgate"
)

func main() {
	action := flag.String("action", "", "install or uninstall")
	pamService := flag.String("pam-service", "/etc/pam.d/gdm-password", "graphical PAM service file")
	helper := flag.String("helper", "/usr/libexec/tempo-pam-check", "installed PAM helper")
	configPath := flag.String("config", "/etc/tempo-agent/config.toml", "agent configuration")
	flag.Parse()

	var err error
	switch *action {
	case "install":
		settings, loadErr := config.Load(*configPath)
		if loadErr != nil {
			err = loadErr
			break
		}
		err = pamgate.Install(pamgate.InstallOptions{
			PAMServicePath: *pamService,
			HelperPath:     *helper,
			ConfigPath:     *configPath,
			ControlledUser: settings.ControlledUser,
		})
	case "uninstall":
		err = pamgate.Uninstall(*pamService)
	default:
		err = fmt.Errorf("action must be install or uninstall")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "tempo-pam-setup: %v\n", err)
		os.Exit(1)
	}
}
