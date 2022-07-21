// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package feature

// FilterFunc is the function use to filter elements from a bundle.
type FilterFunc = func(Featurable) bool

// Bundleable merges featurable and bundle interface together.
type bundleable interface {
	Features() []Featurable
}

// Bundle defines a list of features available in the current beat.
type Bundle struct {
	features []Featurable
}

// NewBundle creates a new Bundle of feature to be registered.
func NewBundle(features ...Featurable) *Bundle {
	return &Bundle{features: features}
}

// FilterWith takes a predicate and return a list of filtered bundle matching the predicate.
func (b *Bundle) FilterWith(pred FilterFunc) *Bundle {
	var filtered []Featurable

	for _, feature := range b.features {
		if pred(feature) {
			filtered = append(filtered, feature)
		}
	}
	return NewBundle(filtered...)
}

// Filter creates a new bundle with only the feature matching the requested stability.
func (b *Bundle) Filter(stabilities ...Stability) *Bundle {
	return b.FilterWith(HasStabilityPred(stabilities...))
}

// Features returns the interface features slice so
func (b *Bundle) Features() []Featurable {
	return b.features
}

// MustBundle takes existing bundle or features and create a new Bundle with all the merged Features.
func MustBundle(bundle ...bundleable) *Bundle {
	var merged []Featurable
	for _, feature := range bundle {
		merged = append(merged, feature.Features()...)
	}
	return NewBundle(merged...)
}

// HasStabilityPred returns true if the feature match any of the provided stabilities.
func HasStabilityPred(stabilities ...Stability) FilterFunc {
	return func(f Featurable) bool {
		for _, s := range stabilities {
			if s == f.Description().Stability {
				return true
			}
		}
		return false
	}
}
