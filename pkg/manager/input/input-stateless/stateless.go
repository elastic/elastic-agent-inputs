// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stateless

import (
	"fmt"
	"runtime/debug"

	"github.com/elastic/go-concert/unison"

	"github.com/elastic/elastic-agent-inputs/pkg/manager/input"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// InputManager provides an InputManager for transient inputs, that do not store
// state in the registry or require end-to-end event acknowledgement.
type InputManager struct {
	Configure func(*conf.C) (Input, error)
}

// Input is the interface transient inputs are required to implemented.
type Input interface {
	Name() string
	Test(input.TestContext) error
	Run(ctx input.Context, publish Publisher) error
}

// Publisher is used by the Input to emit events.
type Publisher interface {
	Publish(publisher.Event)
}

type configuredInput struct {
	input Input
}

var _ input.InputManager = InputManager{}

// NewInputManager wraps the given configure function to create a new stateless input manager.
func NewInputManager(configure func(*conf.C) (Input, error)) InputManager {
	return InputManager{Configure: configure}
}

// Init does nothing. Init is required to fullfil the input.InputManager interface.
func (m InputManager) Init(_ unison.Group, _ input.Mode) error { return nil }

// Create configures a transient input and ensures that the final input can be used with
// with the filebeat input architecture.
func (m InputManager) Create(cfg *conf.C) (input.Input, error) {
	inp, err := m.Configure(cfg)
	if err != nil {
		return nil, err
	}
	return configuredInput{inp}, nil
}

func (si configuredInput) Name() string { return si.input.Name() }

func (si configuredInput) Run(ctx input.Context, pipeline publisher.PipelineConnector) (err error) {
	defer func() {
		if v := recover(); v != nil {
			if e, ok := v.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("input panic with: %+v\n%s", v, debug.Stack())
			}
		}
	}()

	client, err := pipeline.ConnectWith(publisher.ClientConfig{
		PublishMode: publisher.DefaultGuarantees,

		// configure pipeline to disconnect input on stop signal.
		CloseRef: ctx.Cancelation,
	})
	if err != nil {
		return err
	}

	defer client.Close()
	return si.input.Run(ctx, client)
}

func (si configuredInput) Test(ctx input.TestContext) error {
	return si.input.Test(ctx)
}
