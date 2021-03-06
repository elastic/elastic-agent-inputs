// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package inputest

import (
	"errors"

	"github.com/elastic/elastic-agent-inputs/pkg/feature"
	"github.com/elastic/elastic-agent-inputs/pkg/manager/input"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/unison"
)

// MockInputManager can be used as InputManager replacement in tests that require a new Input Manager.
// The OnInit and OnConfigure functions are executed if the corresponding methods get called.
type MockInputManager struct {
	OnInit      func(input.Mode) error
	OnConfigure InputConfigurer
}

// InputConfigurer describes the interface for user supplied functions, that is
// used to create a new input from a configuration object.
type InputConfigurer func(*conf.C) (input.Input, error)

// MockInput can be used as an Input instance in tests that require a new Input with definable behavior.
// The OnTest and OnRun functions are executed if the corresponding methods get called.
type MockInput struct {
	Type   string
	OnTest func(input.TestContext) error
	OnRun  func(input.Context, publisher.PipelineConnector) error
}

// Init returns nil if OnInit is not set. Otherwise the return value of OnInit is returned.
func (m *MockInputManager) Init(_ unison.Group, mode input.Mode) error {
	if m.OnInit != nil {
		return m.OnInit(mode)
	}
	return nil
}

// Create fails with an error if OnConfigure is not set. Otherwise the return
// values of OnConfigure are returned.
func (m *MockInputManager) Create(cfg *conf.C) (input.Input, error) {
	if m.OnConfigure != nil {
		return m.OnConfigure(cfg)
	}
	return nil, errors.New("oops, OnConfigure not implemented ")
}

// Name return the `Type` field of MockInput. It is required to satisfy the input.Input interface.
func (f *MockInput) Name() string { return f.Type }

// Test return nil if OnTest is not set. Otherwise OnTest will be called.
func (f *MockInput) Test(ctx input.TestContext) error {
	if f.OnTest != nil {
		return f.OnTest(ctx)
	}
	return nil
}

// Run returns nil if OnRun is not set.
func (f *MockInput) Run(ctx input.Context, pipeline publisher.PipelineConnector) error {
	if f.OnRun != nil {
		return f.OnRun(ctx, pipeline)
	}
	return nil
}

// ConstInputManager create a MockInputManager that always returns input when
// Configure is called. Use ConstInputManager for tests that require an
// InputManager, but create only one Input instance.
func ConstInputManager(input input.Input) *MockInputManager {
	return &MockInputManager{OnConfigure: ConfigureConstInput(input)}
}

// ConfigureConstInput return an InputConfigurer that returns always input when called.
func ConfigureConstInput(i input.Input) InputConfigurer {
	return func(_ *conf.C) (input.Input, error) {
		return i, nil
	}
}

// SinglePlugin wraps an InputManager into a slice of input.Plugin, that can be used directly with input.NewLoader.
func SinglePlugin(name string, manager input.InputManager) []input.Plugin {
	return []input.Plugin{{
		Name:      name,
		Stability: feature.Stable,
		Manager:   manager,
	}}
}
