package hatchcert

import (
	"os"
	"os/exec"
)

func Hook(cmd []string) error {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
