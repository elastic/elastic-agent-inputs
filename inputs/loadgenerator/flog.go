// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/rs/xid"
)

const JSONLogFormat = `{"@timestamp":"%s", "xid":"%s", "message": "%s", "host":"%s", "user-identifier":"%s", "method": "%s", "request": "%s", "protocol":"%s", "status":%d, "bytes":%d, "referer": "%s"}`

// NewJSONLogFormat creates a log string with json log format
func NewJSONLogFormat(t time.Time) string {
	return fmt.Sprintf(
		JSONLogFormat,
		t.Format(time.RFC3339Nano),
		xid.NewWithTime(t).String(), // xid is sortable by time.
		gofakeit.HackerPhrase(),
		gofakeit.IPv4Address(),
		strings.ToLower(gofakeit.Username()),
		gofakeit.HTTPMethod(),
		RandResourceURI(),
		RandHTTPVersion(),
		gofakeit.HTTPStatusCode(),
		gofakeit.Number(0, 30000),
		gofakeit.URL(),
	)
}

// RandResourceURI generates a random resource URI
func RandResourceURI() string {
	var uri string
	num := gofakeit.Number(1, 4)
	for i := 0; i < num; i++ {
		uri += "/" + url.QueryEscape(gofakeit.BS())
	}
	uri = strings.ToLower(uri)
	return uri
}

// RandHTTPVersion returns a random http version
func RandHTTPVersion() string {
	versions := []string{"HTTP/1.0", "HTTP/1.1", "HTTP/2.0"}
	return versions[rand.Intn(3)] //nolint: gosec // not a security feature
}
