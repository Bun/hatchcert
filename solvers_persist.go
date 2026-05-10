package hatchcert

import (
	"context"

	"github.com/mholt/acmez/v3/acme"
)

// dnsPersist01 does nothing, because the DNS-PERSIST-01 mechanism works with
// preconfigured DNS entries.
// See: https://letsencrypt.org/2026/02/18/dns-persist-01
type dnsPersist01 struct {
}

func (dnsPersist01) Present(ctx context.Context, challenge acme.Challenge) error {
	return nil
}

func (dnsPersist01) CleanUp(ctx context.Context, challenge acme.Challenge) error {
	return nil
}
