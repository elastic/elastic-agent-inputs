// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package input

import (
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/unison"
)

type simpleInputManager struct {
	configure func(*config.C) (Input, error)
}

// ConfigureWith creates an InputManager that provides no extra logic and
// allows each input to fully control event collection and publishing in
// isolation. The function fn will be called for every input to be configured.
func ConfigureWith(fn func(cfc *config.C) (Input, error)) InputManager {
	return &simpleInputManager{configure: fn}
}

// Init is required to fulfil the input.InputManager interface.
// For the kafka input no special initialization is required.
func (*simpleInputManager) Init(grp unison.Group, m Mode) error { return nil }

// Create builds a new Input instance from the given configuration, or returns
// an error if the configuration is invalid.
func (manager *simpleInputManager) Create(cfg *config.C) (Input, error) {
	return manager.configure(cfg)
}
