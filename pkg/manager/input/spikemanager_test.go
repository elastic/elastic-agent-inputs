package input_test

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-inputs/inputs/loadgenerator"
	"github.com/elastic/elastic-agent-inputs/pkg/manager/input"
	"github.com/elastic/elastic-agent-inputs/pkg/outputs/console"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher/acker"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

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
	streamCount := runtime.NumCPU()
	for i := 0; i < streamCount; i++ {
		stream := &proto.Stream{
			Id: fmt.Sprintf("test-stream-%d", i),
			Source: RequireNewStruct(
				map[string]interface{}{
					"delay":       "0s",
					"timedelta":   "1s",
					"currenttime": false,
					"eventscount": 1000,
					"loop":        false,
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
	runtime := time.Millisecond * 50

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

	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	client := client.NewV2(fmt.Sprintf(":%d", srv.Port), token, client.VersionInfo{
		Name:    t.Name(),
		Version: "v1.0.0",
	}, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// Make the logger discard messages because they're too verbose
	logCfg := logp.Config{}
	logp.ToDiscardOutput()(&logCfg)
	logp.Configure(logCfg)

	spkm := input.NewSpikeManager(
		logp.L(),
		client,
		discardOutput(t),
		loadgenerator.NewLoadGenerator,
	)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), runtime*2)
	defer cancel()

	t.Log("Starting client...")
	spkm.Init(timeoutCtx)
	require.NoError(t, err)

	assert.True(t, allHealthy, "loadGenerator units did not report healthy")
	assert.False(t, gotFailure, "LoadGenerator returned a failure")
	assert.True(t, gotFirstCheckin, "Never got checkin from LoadGenerator")
	assert.True(t, gotCorrectStop, "Never got Unit Stopped")
}

// RequireNewStruct converts a mapstr to a protobuf struct
func RequireNewStruct(v map[string]interface{}) *structpb.Struct {
	str, err := structpb.NewStruct(v)
	if err != nil {
		panic(err)
	}
	return str
}

func discardOutput(t *testing.T) publisher.PipelineV2 {
	t.Helper()

	acker := acker.NoOp()
	cfg := console.Config{Enabled: true}

	return console.New(
		context.Background(),
		io.Discard,
		logp.L().Named("discard_output"),
		acker,
		cfg,
	)
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
