package hatchcert

import (
	"context"
	"fmt"

	"github.com/mholt/acmez/v3"
	"github.com/mholt/acmez/v3/acme"
)

type multisolver struct {
	lookup map[string]acmez.Solver
}

func newMultisolver() multisolver {
	return multisolver{lookup: make(map[string]acmez.Solver)}
}

func (m multisolver) Present(ctx context.Context, challenge acme.Challenge) error {
	s, ok := m.lookup[challenge.Identifier.Value]
	if !ok {
		return fmt.Errorf("Present: unconfigured domain %q", challenge.Identifier.Value)
	}
	return s.Present(ctx, challenge)
}

func (m multisolver) CleanUp(ctx context.Context, challenge acme.Challenge) error {
	s, ok := m.lookup[challenge.Identifier.Value]
	if !ok {
		return fmt.Errorf("CleanUp: unconfigured domain %q", challenge.Identifier.Value)
	}
	return s.CleanUp(ctx, challenge)
}

func (m multisolver) Wait(ctx context.Context, challenge acme.Challenge) error {
	s, ok := m.lookup[challenge.Identifier.Value]
	if !ok {
		return fmt.Errorf("Wait: unconfigured domain %q", challenge.Identifier.Value)
	}
	if w, ok := s.(acmez.Waiter); ok {
		return w.Wait(ctx, challenge)
	}
	return nil
}
