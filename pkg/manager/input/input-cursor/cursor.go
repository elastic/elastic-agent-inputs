// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

// Cursor allows the input to check if cursor status has been stored
// in the past and unpack the status into a custom structure.
type Cursor struct {
	store    *store
	resource *resource
}

func makeCursor(store *store, res *resource) Cursor {
	return Cursor{store: store, resource: res}
}

// IsNew returns true if no cursor information has been stored
// for the current Source.
func (c Cursor) IsNew() bool { return c.resource.IsNew() }

// Unpack deserialized the cursor state into to. Unpack fails if no pointer is
// given, or if the structure to points to is not compatible with the document
// stored.
func (c Cursor) Unpack(to interface{}) error {
	if c.IsNew() {
		return nil
	}
	return c.resource.UnpackCursor(to)
}
