package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	fmt.Println("hello world")

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

	streamCount := 1 //runtime.NumCPU()
	for i := 0; i < streamCount; i++ {
		stream := &proto.Stream{
			Id: fmt.Sprintf("test-stream-%d", i),
			Source: RequireNewStruct(
				map[string]interface{}{
					"delay":       "1.0s",
					"timedelta":   "1.0s",
					"currenttime": false,
					// "eventscount": 1000,
					"loop": true,
					// "outfile": filepath.Join("/tmp", fmt.Sprintf("testout-%d.log", i)),
				},
			),
		}
		expectedLoadGen.Streams = append(expectedLoadGen.Streams, stream)
	}

	mut := sync.Mutex{}

	unitOneID := mock.NewID()
	token := "3dde0396-7b28-4946-b2af-af4cc143232d"

	start := time.Now()
	runtime := 24 * time.Hour

	fmt.Println(token)

	srv := mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			mut.Lock()
			defer mut.Unlock()
			if observed.Token == token {
				// initial checkin
				if len(observed.Units) == 0 || observed.Units[0].State == proto.State_STARTING {
					fmt.Println("Sending initial client config...")
					return createUnitsWithState(proto.State_HEALTHY, expectedLoadGen, unitOneID, 0)
				} else if checkUnitStateHealthy(observed.Units) {
					if time.Since(start) > runtime {
						fmt.Println("Sending stopped config...")
						//remove the units once they've been healthy for a given period of time
						return createUnitsWithState(proto.State_STOPPING, expectedLoadGen, unitOneID, 1)
					}
					//otherwise, just remove the units
				} else if observed.Units[0].State == proto.State_STOPPED {
					fmt.Println("Got Unit stopped")
					return &proto.CheckinExpected{
						Units: nil,
					}
				} else if observed.Units[0].State == proto.State_FAILED {
					fmt.Printf("Got Unit Failed: %#v\n", observed.Units[0].Message)
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

	srv.Port = 3000
	if err := srv.Start(); err != nil {
		panic(err)
	}

	defer srv.Stop()

	exit := make(chan os.Signal)
	signal.Notify(exit, os.Interrupt)

	fmt.Println("server running on port:", srv.Port)
	fmt.Println("waiting for a Ctrl + C")
	<-exit
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

// RequireNewStruct converts a mapstr to a protobuf struct
func RequireNewStruct(v map[string]interface{}) *structpb.Struct {
	str, err := structpb.NewStruct(v)
	if err != nil {
		panic(err)
	}
	return str
}
