package hatchcert

import (
	"os"
	"os/exec"
	"strings"
)

func Hook(hook string) error {
	cmd := strings.Split(hook, " ") // TODO: maybe parse this more intelligently
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
