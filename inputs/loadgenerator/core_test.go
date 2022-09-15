// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestSetConfigValues(t *testing.T) {
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

	cfg, err := configFromProtobuf(testStruct)
	require.NoError(t, err)
	t.Logf("Got config: %#v", cfg)
	assert.Equal(t, cfg.Loop, loop)
	assert.Equal(t, cfg.EventsCount, count)
}

func TestWithMockServer(t *testing.T) {
	var expectedLoadGen = &proto.UnitExpectedConfig{
		DataStream: &proto.DataStream{
			Namespace: "default",
		},
		Type:     "input/loadgenerator",
		Id:       "input/loadgenerator",
		Name:     "test-integration-1",
		Revision: 1,
		Meta: &proto.Meta{
			Package: &proto.Package{
				Name:    "load-generator",
				Version: "0.0.0",
			},
		},
	}

	// iterate N times, making N streams of the loadgenerator to test
	strpath, err := os.MkdirTemp("", "test-loadgen")
	defer os.RemoveAll(strpath)

	t.Logf("Writing logs to %s", strpath)
	require.NoError(t, err)
	streamCount := runtime.NumCPU()
	for i := 0; i < streamCount; i++ {
		stream := &proto.Stream{
			Id: fmt.Sprintf("test-stream-%d", i),
			Source: RequireNewStruct(t,
				map[string]interface{}{
					"delay":       "0s",
					"timedelta":   "1s",
					"currenttime": false,
					"eventscount": 1000,
					"loop":        false,
					"outfile":     filepath.Join(strpath, fmt.Sprintf("testout-%d.log", i)),
				},
			),
		}
		expectedLoadGen.Streams = append(expectedLoadGen.Streams, stream)
	}
	t.Logf("Running test with %d load generators", streamCount)

	unitOneID := mock.NewID()
	token := mock.NewID()
	var mut sync.Mutex
	start := time.Now()
	runtime := time.Second * 20

	// checks for the test
	allHealthy := false
	gotFailure := false
	gotFirstCheckin := false
	gotCorrectStop := false

	srv := mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			mut.Lock()
			defer mut.Unlock()
			if observed.Token == token {
				// initial checkin
				if len(observed.Units) == 0 || observed.Units[0].State == proto.State_STARTING {
					t.Logf("Sending initial client config...")
					gotFirstCheckin = true
					return createUnitsWithState(proto.State_HEALTHY, expectedLoadGen, unitOneID, 0)
				} else if checkUnitStateHealthy(observed.Units) {
					allHealthy = true
					if time.Since(start) > runtime {
						t.Logf("Sending stopped config...")
						//remove the units once they've been healthy for a given period of time
						return createUnitsWithState(proto.State_STOPPING, expectedLoadGen, unitOneID, 1)
					}
					//otherwise, just remove the units
				} else if observed.Units[0].State == proto.State_STOPPED {
					t.Logf("Got Unit stopped")
					gotCorrectStop = true
					return &proto.CheckinExpected{
						Units: nil,
					}
				} else if observed.Units[0].State == proto.State_FAILED {
					t.Logf("Got Unit Failed: %#v", observed.Units[0].Message)
					gotFailure = true
					return &proto.CheckinExpected{
						Units: nil,
					}
				}

			}

			return nil
		},
		ActionImpl: func(response *proto.ActionResponse) error {
			return nil
		},
		ActionsChan: make(chan *mock.PerformAction, 100),
	} // end of srv declaration

	err = srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	client := client.NewV2(fmt.Sprintf(":%d", srv.Port), token, client.VersionInfo{
		Name:    "program",
		Version: "v1.0.0",
	}, grpc.WithTransportCredentials(insecure.NewCredentials()))

	_ = logp.DevelopmentSetup()
	lg := newLoadGenerator(logp.L())
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*40)
	defer cancel()

	t.Logf("Starting client...")
	err = lg.StartWithClient(timeoutCtx, client)
	require.NoError(t, err)

	assert.True(t, allHealthy, "loadGenerator units did not report healthy")
	assert.False(t, gotFailure, "LoadGenerator returned a failure")
	assert.True(t, gotFirstCheckin, "Never got checkin from LoadGenerator")
	assert.True(t, gotCorrectStop, "Never got Unit Stopped")
}

func createUnitsWithState(state proto.State, input *proto.UnitExpectedConfig, inID string, stateIndex uint64) *proto.CheckinExpected {
	return &proto.CheckinExpected{
		AgentInfo: &proto.CheckinAgentInfo{
			Id:       "test-agent",
			Version:  "8.4.0",
			Snapshot: true,
		},
		Units: []*proto.UnitExpected{
			{
				Id:             inID,
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: stateIndex,
				Config:         input,
				State:          state,
				LogLevel:       proto.UnitLogLevel_DEBUG,
			},
		},
	}
}

func checkUnitStateHealthy(units []*proto.UnitObserved) bool {
	for _, unit := range units {
		if unit.State != proto.State_HEALTHY {
			return false
		}
	}
	return true
}

//RequireNewStruct converts a mapstr to a protobuf struct
func RequireNewStruct(t *testing.T, v map[string]interface{}) *structpb.Struct {
	str, err := structpb.NewStruct(v)
	require.NoError(t, err)
	return str
}
