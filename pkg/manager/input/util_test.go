// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package input

import (
	"errors"
	"testing"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/unison"
)

type fakeInputManager struct {
	OnInit      func(Mode) error
	OnConfigure func(*conf.C) (Input, error)
}

type fakeInput struct {
	Type   string
	OnTest func(TestContext) error
	OnRun  func(Context, publisher.PipelineConnector) error
}

func makeConfigFakeInput(prototype fakeInput) func(*conf.C) (Input, error) {
	return func(cfg *conf.C) (Input, error) {
		tmp := prototype
		return &tmp, nil
	}
}

func (m *fakeInputManager) Init(_ unison.Group, mode Mode) error {
	if m.OnInit != nil {
		return m.OnInit(mode)
	}
	return nil
}

func (m *fakeInputManager) Create(cfg *conf.C) (Input, error) {
	if m.OnConfigure != nil {
		return m.OnConfigure(cfg)
	}
	return nil, errors.New("oops")
}

func (f *fakeInput) Name() string { return f.Type }
func (f *fakeInput) Test(ctx TestContext) error {
	if f.OnTest != nil {
		return f.OnTest(ctx)
	}
	return nil
}

func (f *fakeInput) Run(ctx Context, pipeline publisher.PipelineConnector) error {
	if f.OnRun != nil {
		return f.OnRun(ctx, pipeline)
	}
	return nil
}

func expectError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error")
	}
}

func expectNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
