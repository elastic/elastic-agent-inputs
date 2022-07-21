// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package inputest

import (
	"testing"

	"github.com/elastic/elastic-agent-inputs/pkg/manager/input"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Loader wraps the input Loader in order to provide additional methods for reuse in tests.
type Loader struct {
	t testing.TB
	*input.Loader
}

// MustNewTestLoader creates a new Loader. The test fails with fatal if the
// NewLoader constructor function returns an error.
func MustNewTestLoader(t testing.TB, plugins []input.Plugin, typeField, defaultType string) *Loader {
	l, err := input.NewLoader(logp.NewLogger("test"), plugins, typeField, defaultType)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}
	return &Loader{t: t, Loader: l}
}

// MustConfigure confiures a new input. The test fails with t.Fatal if the
// operation failed.
func (l *Loader) MustConfigure(cfg *conf.C) input.Input {
	i, err := l.Configure(cfg)
	if err != nil {
		l.t.Fatalf("Failed to create the input: %v", err)
	}
	return i
}
