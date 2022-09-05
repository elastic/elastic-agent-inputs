// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-inputs/pkg/manager/input"
	"github.com/elastic/elastic-agent-inputs/pkg/manager/internal/resources"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	pubtest "github.com/elastic/elastic-agent-inputs/pkg/publisher/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-shipper-client/pkg/helpers"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
	"github.com/elastic/go-concert/unison"
)

type fakeTestInput struct {
	OnTest func(Source, input.TestContext) error
	OnRun  func(input.Context, Source, Cursor, Publisher) error
}

type stringSource string

func TestManager_Init(t *testing.T) {
	// Integration style tests for the InputManager and the state garbage collector

	t.Run("stopping the taskgroup kills internal go-routines", func(t *testing.T) {
		numRoutines := runtime.NumGoroutine()

		var grp unison.TaskGroup
		store := createSampleStore(t, nil)
		manager := &InputManager{
			Logger:              logp.NewLogger("test"),
			StateStore:          store,
			Type:                "test",
			DefaultCleanTimeout: 10 * time.Millisecond,
		}

		err := manager.Init(&grp, input.ModeRun)
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)
		_ = grp.Stop()

		// wait for all go-routines to be gone

		for numRoutines < runtime.NumGoroutine() {
			time.Sleep(1 * time.Millisecond)
		}
	})

	t.Run("collect old entries after startup", func(t *testing.T) {
		store := createSampleStore(t, map[string]state{
			"test::key": {
				TTL:     1 * time.Millisecond,
				Updated: time.Now().Add(-24 * time.Hour),
			},
		})
		store.GCPeriod = 10 * time.Millisecond

		var grp unison.TaskGroup
		defer func() {
			_ = grp.Stop()
		}()
		manager := &InputManager{
			Logger:              logp.NewLogger("test"),
			StateStore:          store,
			Type:                "test",
			DefaultCleanTimeout: 10 * time.Millisecond,
		}

		err := manager.Init(&grp, input.ModeRun)
		require.NoError(t, err)

		for len(store.snapshot()) > 0 {
			time.Sleep(1 * time.Millisecond)
		}
	})
}

func TestManager_Create(t *testing.T) {
	t.Run("fail if no source is configured", func(t *testing.T) {
		manager := constInput(t, nil, &fakeTestInput{})
		_, err := manager.Create(conf.NewConfig())
		require.Error(t, err)
	})

	t.Run("fail if config error", func(t *testing.T) {
		manager := failingManager(t, errors.New("oops"))
		_, err := manager.Create(conf.NewConfig())
		require.Error(t, err)
	})

	t.Run("fail if no input runner is returned", func(t *testing.T) {
		manager := constInput(t, sourceList("test"), nil)
		_, err := manager.Create(conf.NewConfig())
		require.Error(t, err)
	})

	t.Run("configure ok", func(t *testing.T) {
		manager := constInput(t, sourceList("test"), &fakeTestInput{})
		_, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)
	})

	t.Run("configuring inputs with overlapping sources is allowed", func(t *testing.T) {
		manager := simpleManagerWithConfigure(t, func(cfg *conf.C) ([]Source, Input, error) {
			config := struct{ Sources []string }{}
			err := cfg.Unpack(&config)
			return sourceList(config.Sources...), &fakeTestInput{}, err
		})

		_, err := manager.Create(conf.MustNewConfigFrom(map[string]interface{}{
			"sources": []string{"a"},
		}))
		require.NoError(t, err)

		_, err = manager.Create(conf.MustNewConfigFrom(map[string]interface{}{
			"sources": []string{"a"},
		}))
		require.NoError(t, err)
	})
}

func TestManager_InputsTest(t *testing.T) {
	var mu sync.Mutex
	var seen []string

	sources := sourceList("source1", "source2")

	t.Run("test is run for each source", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(source Source, _ input.TestContext) error {
				mu.Lock()
				defer mu.Unlock()
				seen = append(seen, source.Name())
				return nil
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		err = inp.Test(input.TestContext{})
		require.NoError(t, err)

		sort.Strings(seen)
		require.Equal(t, []string{"source1", "source2"}, seen)
	})

	t.Run("cancel gets distributed to all source tests", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(_ Source, ctx input.TestContext) error {
				<-ctx.Cancelation.Done()
				return nil
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.TODO())

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Test(input.TestContext{Cancelation: ctx})
		}()

		cancel()
		wg.Wait()
		require.NoError(t, err)
	})

	t.Run("fail if test for one source fails", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		failing := Source(stringSource("source1"))
		sources := []Source{failing, stringSource("source2")}

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(source Source, _ input.TestContext) error {
				if source == failing {
					t.Log("return error")
					return errors.New("oops")
				}
				t.Log("return ok")
				return nil
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Test(input.TestContext{})
			t.Logf("Test returned: %v", err)
		}()

		wg.Wait()
		require.Error(t, err)
	})

	t.Run("panic is captured", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(source Source, _ input.TestContext) error {
				panic("oops")
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Test(input.TestContext{Logger: logp.NewLogger("test")})
			t.Logf("Test returned: %v", err)
		}()

		wg.Wait()
		require.Error(t, err)
	})
}

func TestManager_InputsRun(t *testing.T) {
	// Integration style tests for the InputManager and Input.Run

	t.Run("input returned with error", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sourceList("test"), &fakeTestInput{
			OnRun: func(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
				return errors.New("oops")
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var clientCounters pubtest.ClientCounter
		err = inp.Run(input.Context{
			Logger:      manager.Logger,
			Cancelation: cancelCtx,
		}, clientCounters.BuildConnector())
		require.Error(t, err)
		require.Equal(t, 0, clientCounters.Active())
	})

	t.Run("panic is captured", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sourceList("test"), &fakeTestInput{
			OnRun: func(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
				panic("oops")
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var clientCounters pubtest.ClientCounter
		err = inp.Run(input.Context{
			Logger:      manager.Logger,
			Cancelation: cancelCtx,
		}, clientCounters.BuildConnector())
		require.Error(t, err)
		require.Equal(t, 0, clientCounters.Active())
	})

	t.Run("shutdown on signal", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sourceList("test"), &fakeTestInput{
			OnRun: func(ctx input.Context, _ Source, _ Cursor, _ Publisher) error {
				<-ctx.Cancelation.Done()
				return nil
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var clientCounters pubtest.ClientCounter
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Run(input.Context{
				Logger:      manager.Logger,
				Cancelation: cancelCtx,
			}, clientCounters.BuildConnector())
		}()

		cancel()
		wg.Wait()
		require.NoError(t, err)
		require.Equal(t, 0, clientCounters.Active())
	})

	t.Run("continue sending from last known position", func(t *testing.T) {
		log := logp.NewLogger("test")

		type runConfig struct{ Max int }

		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()

		manager := simpleManagerWithConfigure(t, func(cfg *conf.C) ([]Source, Input, error) {
			config := runConfig{}
			if err := cfg.Unpack(&config); err != nil {
				return nil, nil, err
			}

			inp := &fakeTestInput{
				OnRun: func(_ input.Context, _ Source, cursor Cursor, pub Publisher) error {
					state := struct{ N int }{}
					if !cursor.IsNew() {
						if err := cursor.Unpack(&state); err != nil {
							return fmt.Errorf("failed to unpack cursor: %w", err)
						}
					}

					for i := 0; i < config.Max; i++ {
						event := publisher.Event{
							ShipperMessage: &messages.Event{
								Fields: &messages.Struct{
									Data: map[string]*messages.Value{
										"n": helpers.NewNumberValue(float64(state.N)),
									},
								},
							},
						}

						state.N++
						mustPublish(pub, event, state)
					}
					return nil
				},
			}

			return sourceList("test"), inp, nil
		})

		var ids []int
		pipeline := pubtest.ConstClient(&pubtest.FakeClient{
			PublishFunc: func(event publisher.Event) {
				fields := event.ShipperMessage.Fields.GetData()
				id, ok := fields["n"]
				if !ok {
					t.Fatalf("cannot get key 'n' from: %#v", fields)
				}
				ids = append(ids, int(id.GetNumberValue()))
			},
		})

		// create and run first instance
		inp, err := manager.Create(conf.MustNewConfigFrom(runConfig{Max: 3}))
		require.NoError(t, err)
		require.NoError(t, inp.Run(input.Context{
			Logger:      log,
			Cancelation: context.Background(),
		}, pipeline))

		// create and run second instance instance
		inp, err = manager.Create(conf.MustNewConfigFrom(runConfig{Max: 3}))
		require.NoError(t, err)
		err = inp.Run(input.Context{
			Logger:      log,
			Cancelation: context.Background(),
		}, pipeline)
		assert.NoError(t, err)

		// verify
		assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, ids)
	})

	t.Run("event ACK triggers execution of update operations", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		store := createSampleStore(t, nil)
		var wgSend sync.WaitGroup
		wgSend.Add(1)
		manager := constInput(t, sourceList("key"), &fakeTestInput{
			OnRun: func(ctx input.Context, _ Source, _ Cursor, pub Publisher) error {
				defer wgSend.Done()
				fields := &messages.Event{
					Fields: &messages.Struct{
						Data: map[string]*messages.Value{
							"hello": helpers.NewStringValue("world"),
						},
					},
				}
				mustPublish(pub, publisher.Event{ShipperMessage: fields}, "test-cursor-state1")
				mustPublish(pub, publisher.Event{ShipperMessage: fields}, "test-cursor-state2")
				mustPublish(pub, publisher.Event{ShipperMessage: fields}, "test-cursor-state3")
				mustPublish(pub, publisher.Event{ShipperMessage: fields}, nil)
				mustPublish(pub, publisher.Event{ShipperMessage: fields}, "test-cursor-state4")
				mustPublish(pub, publisher.Event{ShipperMessage: fields}, "test-cursor-state5")
				mustPublish(pub, publisher.Event{ShipperMessage: fields}, "test-cursor-state6")
				return nil
			},
		})
		manager.StateStore = store

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// setup publishing pipeline and capture ACKer, so we can simulate progress in the Output
		var acker publisher.ACKer
		var wgACKer sync.WaitGroup
		wgACKer.Add(1)
		pipeline := &pubtest.FakeConnector{
			ConnectFunc: func(cfg publisher.ClientConfig) (publisher.Client, error) {
				defer wgACKer.Done()
				acker = cfg.ACKHandler
				return &pubtest.FakeClient{
					PublishFunc: func(event publisher.Event) {
						acker.AddEvent(event, true)
					},
				}, nil
			},
		}

		// start the input
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Run(input.Context{
				Logger:      manager.Logger,
				Cancelation: cancelCtx,
			}, pipeline)
		}()
		// wait for test setup to shutdown
		defer wg.Wait()

		// wait for setup complete and events being send (pending operations in the pipeline)
		wgACKer.Wait()
		wgSend.Wait()

		// 1. No cursor state in store yet, all operations are still pending
		require.Equal(t, nil, store.snapshot()["test::key"].Cursor)

		// ACK first 2 events and check snapshot state
		acker.ACKEvents(2)
		require.Equal(t, "test-cursor-state2", store.snapshot()["test::key"].Cursor)

		// ACK 1 events and check snapshot state (3 events published)
		acker.ACKEvents(1)
		require.Equal(t, "test-cursor-state3", store.snapshot()["test::key"].Cursor)

		// ACK event without cursor update and check snapshot state not modified
		acker.ACKEvents(1)
		require.Equal(t, "test-cursor-state3", store.snapshot()["test::key"].Cursor)

		// ACK rest
		acker.ACKEvents(3)
		require.Equal(t, "test-cursor-state6", store.snapshot()["test::key"].Cursor)
	})
}

func mustPublish(p Publisher, e publisher.Event, cursor interface{}) {
	err := p.Publish(e, cursor)
	if err != nil {
		panic(err)
	}
}

func TestLockResource(t *testing.T) {
	t.Run("can lock unused resource", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()

		res := store.Get("test::key")
		err := lockResource(logp.NewLogger("test"), res, context.TODO())
		require.NoError(t, err)
	})

	t.Run("fail to lock resource in use when context is cancelled", func(t *testing.T) {
		log := logp.NewLogger("test")

		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()

		resUsed := store.Get("test::key")
		err := lockResource(log, resUsed, context.TODO())
		require.NoError(t, err)

		// fail to lock resource in use
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()
		resFail := store.Get("test::key")
		err = lockResource(log, resFail, ctx)
		require.Error(t, err)
		resFail.Release()

		// unlock and release resource in use -> it should be marked finished now
		releaseResource(resUsed)
		require.True(t, resUsed.Finished())
	})

	t.Run("succeed to lock resource after it has been released", func(t *testing.T) {
		log := logp.NewLogger("test")

		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()

		resUsed := store.Get("test::key")
		err := lockResource(log, resUsed, context.TODO())
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			resOther := store.Get("test::key")
			err := lockResource(log, resOther, context.TODO())
			if err == nil {
				releaseResource(resOther)
			}
		}()

		go func() {
			time.Sleep(100 * time.Millisecond)
			releaseResource(resUsed)
		}()

		wg.Wait() // <- block forever if waiting go-routine can not acquire lock
	})
}

func (s stringSource) Name() string { return string(s) }

func simpleManagerWithConfigure(t *testing.T, configure func(*conf.C) ([]Source, Input, error)) *InputManager {
	return &InputManager{
		Logger:     logp.NewLogger("test"),
		StateStore: createSampleStore(t, nil),
		Type:       "test",
		Configure:  configure,
	}
}

func constConfigureResult(t *testing.T, sources []Source, inp Input, err error) *InputManager {
	return simpleManagerWithConfigure(t, func(cfg *conf.C) ([]Source, Input, error) {
		return sources, inp, err
	})
}

func failingManager(t *testing.T, err error) *InputManager {
	return constConfigureResult(t, nil, nil, err)
}

func constInput(t *testing.T, sources []Source, inp Input) *InputManager {
	return constConfigureResult(t, sources, inp, nil)
}

func (f *fakeTestInput) Name() string { return "test" }

func (f *fakeTestInput) Test(source Source, ctx input.TestContext) error {
	if f.OnTest != nil {
		return f.OnTest(source, ctx)
	}
	return nil
}

func (f *fakeTestInput) Run(ctx input.Context, source Source, cursor Cursor, pub Publisher) error {
	if f.OnRun != nil {
		return f.OnRun(ctx, source, cursor, pub)
	}
	return nil
}

func sourceList(names ...string) []Source {
	tmp := make([]Source, len(names))
	for i, name := range names {
		tmp[i] = stringSource(name)
	}
	return tmp
}
