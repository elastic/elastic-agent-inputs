// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package input

import (
	"testing"

	"github.com/elastic/elastic-agent-inputs/pkg/feature"
)

func TestPlugin_Validate(t *testing.T) {
	cases := map[string]struct {
		valid  bool
		plugin Plugin
	}{
		"valid": {
			valid: true,
			plugin: Plugin{
				Name:       "test",
				Stability:  feature.Stable,
				Deprecated: false,
				Info:       "test",
				Doc:        "doc string",
				Manager:    ConfigureWith(nil),
			},
		},
		"missing name": {
			valid: false,
			plugin: Plugin{
				Stability:  feature.Stable,
				Deprecated: false,
				Info:       "test",
				Doc:        "doc string",
				Manager:    ConfigureWith(nil),
			},
		},
		"invalid stability": {
			valid: false,
			plugin: Plugin{
				Name:       "test",
				Deprecated: false,
				Info:       "test",
				Doc:        "doc string",
				Manager:    ConfigureWith(nil),
			},
		},
		"missing manager": {
			valid: false,
			plugin: Plugin{
				Name:       "test",
				Stability:  feature.Stable,
				Deprecated: false,
				Info:       "test",
				Doc:        "doc string",
			},
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			err := test.plugin.validate()
			if test.valid {
				expectNoError(t, err)
			} else {
				expectError(t, err)
			}
		})
	}
}
