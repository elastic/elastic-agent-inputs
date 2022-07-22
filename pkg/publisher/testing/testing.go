// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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
