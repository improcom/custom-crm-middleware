package dataschema

import (
	"errors"
	"fmt"
	"strings"
)

type report struct {
	ctx  string
	errs []string
}

func (r *report) new(format string, args ...any) *report {
	r = &report{fmt.Sprintf(format, args...), make([]string, 0)}
	return r
}

func (r *report) error(msgs ...string) error {
	for _, msg := range msgs {
		r.must(errors.New(msg))
	}

	switch len(r.errs) {
	case 0:
		return nil
	case 1:
		return errors.New(r.ctx + ": " + r.errs[0])
	}

	lines := []string{fmt.Sprintf("%s (%d errors):", r.ctx, len(r.errs))}
	for _, msg := range r.errs {
		lines = append(lines, strings.ReplaceAll(msg, "\n", "\n |"))
	}
	return errors.New(strings.Join(lines, "\n ^ "))
}

func (r *report) must(err error) {
	if err != nil {
		r.errs = append(r.errs, err.Error())
	}
}

func (r *report) mustNonEmpty(ctx string, val string) {
	if val == "" {
		r.must(fmt.Errorf("'%s' is required", ctx))
	}
}

func (r *report) mustLen(ctx string, size int) {
	if size < 1 {
		r.must(fmt.Errorf("'%s' cannot be empty", ctx))
	}
}
