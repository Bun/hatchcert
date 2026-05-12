package hatchcert

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/mholt/acmez/v3/acme"
)

// An execSolver provides a solution to the ACME challenge in a subprocess. The
// subprocess is called with the following environment variables set:
//
//     ACMECHAL_ID (FQDN)
//     ACMECHAL_TYPE
//     ACMECHAL_TOKEN
//     ACMECHAL_KEYAUTHZ
//
// Additionally, if the type is `dns-01`:
//
//     ACMECHAL_RECORD
//     ACMECHAL_KEYAUTHZ_SHA256
//
// The process is called with an argument that represents the action: present,
// cleanup, or wait. The process must communicate failure using an exit code.
// Actions outside of `wait` should not block/poll.
type execSolver struct {
	command string
}

func (x execSolver) Present(ctx context.Context, challenge acme.Challenge) error {
	return x.run(ctx, challenge, "present")
}

func (x execSolver) CleanUp(ctx context.Context, challenge acme.Challenge) error {
	return x.run(ctx, challenge, "cleanup")
}

func (x execSolver) Wait(ctx context.Context, challenge acme.Challenge) error {
	return x.run(ctx, challenge, "wait")
}

func (x execSolver) run(ctx context.Context, challenge acme.Challenge, action string) error {
	parts := strings.Fields(x.command)
	cmd := exec.CommandContext(ctx, parts[0], append(parts[1:], action)...)

	env := os.Environ()
	env = append(env,
		"ACMECHAL_ID="+challenge.Identifier.Value,
		"ACMECHAL_TYPE="+challenge.Type,
		"ACMECHAL_TOKEN="+challenge.Token,
		"ACMECHAL_KEYAUTHZ="+challenge.KeyAuthorization,
	)

	if challenge.Type == "dns-01" {
		env = append(env,
			"ACMECHAL_RECORD="+challenge.DNS01TXTRecordName(),
			"ACMECHAL_KEYAUTHZ_SHA256="+challenge.DNS01KeyAuthorization(),
		)
	}

	cmd.Env = env

	for _, pipe := range []struct {
		get func() (io.ReadCloser, error)
		tag string
	}{
		{cmd.StdoutPipe, "stdout"},
		{cmd.StderrPipe, "stderr"},
	} {
		rc, err := pipe.get()
		if err != nil {
			return err
		}
		go func(rc io.ReadCloser, tag string) {
			defer rc.Close()
			scanner := bufio.NewScanner(rc)
			for scanner.Scan() {
				slog.Debug(scanner.Text(),
					"solver", x.command,
					"action", action,
					"stream", tag,
				)
			}
		}(rc, pipe.tag)
	}

	return cmd.Run()
}
