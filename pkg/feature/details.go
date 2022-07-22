// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package feature

import "fmt"

// Details minimal information that you must provide when creating a feature.
type Details struct {
	Name       string
	Stability  Stability
	Deprecated bool
	Info       string // short info string
	Doc        string // long doc string
}

func (d Details) String() string {
	fmtStr := "name: %s, description: %s (%s)"
	if d.Deprecated {
		fmtStr = "name: %s, description: %s (deprecated, %s)"
	}
	return fmt.Sprintf(fmtStr, d.Name, d.Info, d.Stability)
}

// MakeDetails return the minimal information a new feature must provide.
func MakeDetails(fullName string, doc string, stability Stability) Details {
	return Details{Name: fullName, Info: doc, Stability: stability}
}
