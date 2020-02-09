package hatchcert

import "strings"

type MultiError []error

func (m MultiError) Error() string {
	if len(m) == 1 {
		return m[0].Error()
	}
	var errs []string
	for _, e := range m {
		errs = append(errs, e.Error())
	}
	return strings.Join(errs, "; ")
}

func (m MultiError) Nil() error {
	// Go quirk
	if len(m) == 0 {
		return nil
	}
	return m
}
