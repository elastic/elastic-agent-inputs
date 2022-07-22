// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package feature

//go:generate stringer -type=Stability

// Stability defines the stability of the feature, this value can be used to filter a bundler.
type Stability int

// List all the available stability for a feature.
const (
	Undefined Stability = iota
	Stable
	Beta
	Experimental
)
