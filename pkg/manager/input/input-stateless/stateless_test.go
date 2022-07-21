// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stateless_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-inputs/pkg/manager/input"
	stateless "github.com/elastic/elastic-agent-inputs/pkg/manager/input/input-stateless"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	pubtest "github.com/elastic/elastic-agent-inputs/pkg/publisher/testing"
	"github.com/elastic/elastic-agent-libs/atomic"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type fakeStatelessInput struct {
	OnTest func(input.TestContext) error
	OnRun  func(input.Context, stateless.Publisher) error
}

func TestStateless_Run(t *testing.T) {
	t.Run("events are published", func(t *testing.T) {
		const numEvents = 5

		ch := make(chan publisher.Event)
		connector := pubtest.ConstClient(pubtest.ChClient(ch))

		inp := createConfiguredInput(t, constInputManager(&fakeStatelessInput{
			OnRun: func(ctx input.Context, p stateless.Publisher) error {
				defer close(ch)
				for i := 0; i < numEvents; i++ {
					p.Publish(publisher.Event{Fields: map[string]interface{}{"id": i}})
				}
				return nil
			},
		}), nil)

		var err error
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Run(input.Context{}, connector)
		}()

		var receivedEvents int
		for range ch {
			receivedEvents++
		}
		wg.Wait()

		require.NoError(t, err)
		require.Equal(t, numEvents, receivedEvents)
	})

	t.Run("capture panic and return error", func(t *testing.T) {
		inp := createConfiguredInput(t, constInputManager(&fakeStatelessInput{
			OnRun: func(_ input.Context, _ stateless.Publisher) error {
				panic("oops")
			},
		}), nil)

		var clientCounters pubtest.ClientCounter
		err := inp.Run(input.Context{}, clientCounters.BuildConnector())

		require.Error(t, err)
		require.Equal(t, 1, clientCounters.Total())
		require.Equal(t, 0, clientCounters.Active())
	})

	t.Run("publisher unblocks if shutdown signal is send", func(t *testing.T) {
		// the input blocks in the publisher. We loop until the shutdown signal is received
		var started atomic.Bool
		inp := createConfiguredInput(t, constInputManager(&fakeStatelessInput{
			OnRun: func(ctx input.Context, p stateless.Publisher) error {
				for ctx.Cancelation.Err() == nil {
					started.Store(true)
					p.Publish(publisher.Event{
						Fields: mapstr.M{
							"hello": "world",
						},
					})
				}
				return ctx.Cancelation.Err()
			},
		}), nil)

		// connector creates a client the blocks forever until the shutdown signal is received
		var publishCalls atomic.Int
		connector := pubtest.FakeConnector{
			ConnectFunc: func(config publisher.ClientConfig) (publisher.Client, error) {
				return &pubtest.FakeClient{
					PublishFunc: func(event publisher.Event) {
						publishCalls.Inc()
						<-config.CloseRef.Done()
					},
				}, nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup
		var err error
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Run(input.Context{Cancelation: ctx}, connector)
		}()

		// signal and wait for shutdown
		for !started.Load() {
			runtime.Gosched()
		}
		cancel()
		wg.Wait()

		// validate
		require.Equal(t, context.Canceled, err)
		require.Equal(t, 1, publishCalls.Load())
	})

	t.Run("do not start input of pipeline connection fails", func(t *testing.T) {
		errOpps := errors.New("oops")
		connector := pubtest.FailingConnector(errOpps)

		var run atomic.Int
		i := createConfiguredInput(t, constInputManager(&fakeStatelessInput{
			OnRun: func(_ input.Context, publisher stateless.Publisher) error {
				run.Inc()
				return nil
			},
		}), nil)

		err := i.Run(input.Context{}, connector)
		require.True(t, errors.Is(err, errOpps))
		require.Equal(t, 0, run.Load())
	})
}

func (f *fakeStatelessInput) Name() string { return "test" }

func (f *fakeStatelessInput) Test(ctx input.TestContext) error {
	if f.OnTest != nil {
		return f.OnTest(ctx)
	}
	return nil
}

func (f *fakeStatelessInput) Run(ctx input.Context, publish stateless.Publisher) error {
	if f.OnRun != nil {
		return f.OnRun(ctx, publish)
	}
	return errors.New("oops, run not implemented")
}

//nolint:unparam // when we add more tests it will get a config
func createConfiguredInput(t *testing.T, manager stateless.InputManager, config map[string]interface{}) input.Input {
	input, err := manager.Create(conf.MustNewConfigFrom(config))
	require.NoError(t, err)
	return input
}

func constInputManager(input stateless.Input) stateless.InputManager {
	return stateless.NewInputManager(constInput(input))
}

func constInput(input stateless.Input) func(*conf.C) (stateless.Input, error) {
	return func(_ *conf.C) (stateless.Input, error) {
		return input, nil
	}
}
