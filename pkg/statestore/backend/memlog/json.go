// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package memlog

import (
	"io"

	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"
)

type jsonEncoder struct {
	out    io.Writer
	folder *gotype.Iterator
}

func newJSONEncoder(out io.Writer) *jsonEncoder {
	e := &jsonEncoder{out: out}
	e.reset()
	return e
}

func (e *jsonEncoder) reset() {
	visitor := json.NewVisitor(e.out)
	visitor.SetEscapeHTML(false)

	var err error

	// create new encoder with custom time.Time encoding
	e.folder, err = gotype.NewIterator(visitor)
	if err != nil {
		panic(err)
	}
}

func (e *jsonEncoder) Encode(v interface{}) error {
	return e.folder.Fold(v)
}
