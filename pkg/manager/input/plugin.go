// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package input

import (
	"fmt"

	"github.com/elastic/elastic-agent-inputs/pkg/feature"
)

// Plugin describes an input type. Input types should provide a constructor
// function that requires dependencies to be passed and fills out the Plugin structure.
// The Manager is used to finally create and manage inputs of the same type.
// The input-stateless and input-cursor packages, as well as the ConfigureWith function provide
// sample input managers.
//
// Example (stateless input):
//
//   func Plugin() input.Plugin {
//       return input.Plugin{
//           Name: "myservice",
//           Stability: feature.Stable,
//           Deprecated: false,
//           Info: "collect data from myservice",
//           Manager: stateless.NewInputManager(configure),
//       }
//   }
//
type Plugin struct {
	// Name of the input type.
	Name string

	// Configure the input stability. If the stability is not 'Stable' a message
	// is logged when the input type is configured.
	Stability feature.Stability

	// Deprecated marks the plugin as deprecated. If set a deprecation message is logged if
	// an input is configured.
	Deprecated bool

	// Info contains a short description of the input type.
	Info string

	// Doc contains an optional longer description.
	Doc string

	// Manager MUST be configured. The manager is used to create the inputs.
	Manager InputManager
}

// Details returns a generic feature description that is compatible with the
// feature package.
func (p Plugin) Details() feature.Details {
	return feature.Details{
		Name:       p.Name,
		Stability:  p.Stability,
		Deprecated: p.Deprecated,
		Info:       p.Info,
		Doc:        p.Doc,
	}
}

func (p Plugin) validate() error {
	if p.Name == "" {
		return fmt.Errorf("input plugin without name found")
	}
	switch p.Stability {
	case feature.Beta, feature.Experimental, feature.Stable:
		break
	default:
		return fmt.Errorf("plugin '%v' has stability not set", p.Name)
	}
	if p.Manager == nil {
		return fmt.Errorf("invalid plugin (%v) structure detected", p.Name)
	}
	return nil
}
