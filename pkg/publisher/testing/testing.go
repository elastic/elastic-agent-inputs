// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package testing

import "github.com/elastic/elastic-agent-inputs/pkg/publisher"

// ChanClient implements Client interface, forwarding published events to some

type TestPublisher struct {
	client publisher.Client
}

// given channel only.
type ChanClient struct {
	done            chan struct{}
	Channel         chan publisher.Event
	publishCallback func(event publisher.Event)
}

func PublisherWithClient(client publisher.Client) publisher.Pipeline {
	return &TestPublisher{client}
}

func (pub *TestPublisher) Connect() (publisher.Client, error) {
	return pub.client, nil
}

func (pub *TestPublisher) ConnectWith(_ publisher.ClientConfig) (publisher.Client, error) {
	return pub.client, nil
}

func NewChanClientWithCallback(bufSize int, callback func(event publisher.Event)) *ChanClient {
	chanClient := NewChanClientWith(make(chan publisher.Event, bufSize))
	chanClient.publishCallback = callback

	return chanClient
}

func NewChanClient(bufSize int) *ChanClient {
	return NewChanClientWith(make(chan publisher.Event, bufSize))
}

func NewChanClientWith(ch chan publisher.Event) *ChanClient {
	if ch == nil {
		ch = make(chan publisher.Event, 1)
	}
	c := &ChanClient{
		done:    make(chan struct{}),
		Channel: ch,
	}
	return c
}

func (c *ChanClient) Close() error {
	close(c.done)
	return nil
}

// PublishEvent will publish the event on the channel. Options will be ignored.
// Always returns true.
func (c *ChanClient) Publish(event publisher.Event) {
	select {
	case <-c.done:
	case c.Channel <- event:
		if c.publishCallback != nil {
			c.publishCallback(event)
			<-c.Channel
		}
	}
}

func (c *ChanClient) PublishAll(event []publisher.Event) {
	for _, e := range event {
		c.Publish(e)
	}
}

func (c *ChanClient) ReceiveEvent() publisher.Event {
	return <-c.Channel
}
