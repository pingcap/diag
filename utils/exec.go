package utils

import (
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func StartCommands(cmds ...*exec.Cmd) error {
	for _, cmd := range cmds {
		log.Info(cmd.Args)
		if err := cmd.Start(); err != nil {
			return err
		}
	}
	return nil
}

func WaitCommands(cmds ...*exec.Cmd) error {
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func RunCommands(cmds ...*exec.Cmd) error {
	for _, cmd := range cmds {
		log.Info(cmd.Args)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
