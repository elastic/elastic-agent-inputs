// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConfigFromSource(t *testing.T) {
	loop := false
	count := uint64(1000)
	setup := map[string]interface{}{
		"delay":       "1s",
		"timedelta":   "5s",
		"currenttime": true,
		"eventscount": count,
		"loop":        loop,
	}
	testStruct, err := structpb.NewStruct(setup)
	require.NoError(t, err)

	cfg, err := configFromSource(testStruct)
	require.NoError(t, err)
	assert.Equal(t, cfg.Loop, loop)
	assert.Equal(t, cfg.EventsCount, count)
}
