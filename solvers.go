package hatchcert

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mholt/acmez/v3/acme"
)

var ErrUnsafeToken = errors.New("challenge token contains unsafe characters")

//
// HTTP-01 solver with external webserver
//

type webrootSolver struct {
	root string
}

func (s webrootSolver) Present(ctx context.Context, chal acme.Challenge) error {
	if !safePath(chal.Token) {
		return ErrUnsafeToken
	}
	tpath := filepath.Join(s.root, chal.Token)
	slog.Debug("HTTP-01 WebRoot solver: creating token",
		"path", tpath)
	return os.WriteFile(tpath, []byte(chal.KeyAuthorization), 0644)
}

func (s webrootSolver) CleanUp(ctx context.Context, chal acme.Challenge) error {
	if !safePath(chal.Token) {
		return ErrUnsafeToken
	}
	tpath := filepath.Join(s.root, chal.Token)
	slog.Debug("HTTP-01 WebRoot solver: removing token",
		"path", tpath)
	return os.Remove(tpath)
}

//
// HTTP-01 solver with built-in webserver
//

var Mux = http.NewServeMux()

type httpSolver struct {
	listen string
}

func (s httpSolver) Present(ctx context.Context, chal acme.Challenge) error {
	slog.Debug("HTTP-01 HTTP solver: adding token",
		"token", chal.Token)
	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(chal.KeyAuthorization))
		slog.Debug("HTTP-01 HTTP solver: token requested",
			"auth", chal.KeyAuthorization)
	}
	Mux.Handle("/.well-known/acme-challenge/"+chal.Token, http.HandlerFunc(h))
	return nil
}

func (s httpSolver) CleanUp(ctx context.Context, chal acme.Challenge) error {
	// We can't remove paths, so do nothing
	return nil
}
