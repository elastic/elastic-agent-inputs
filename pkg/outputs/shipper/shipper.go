// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package shipper

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	sc "github.com/elastic/elastic-agent-shipper-client/pkg/proto"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
)

type shipper struct {
	cancelCtx context.Context
	logger    *logp.Logger
	conn      *grpc.ClientConn
	client    sc.ProducerClient
	timeout   time.Duration
	config    Config
	acker     publisher.ACKer
}

func New(ctx context.Context, cfg Config, logger *logp.Logger, acker publisher.ACKer) (publisher.PipelineV2, error) {
	s := shipper{
		cancelCtx: ctx,
		logger:    logger,
		acker:     acker,
		config:    cfg,
	}

	tls, err := tlscommon.LoadTLSConfig(s.config.TLS)
	if err != nil {
		return nil, fmt.Errorf("invalid shipper TLS configuration: %w", err)
	}

	var creds credentials.TransportCredentials
	if s.config.TLS != nil && s.config.TLS.Enabled != nil && *s.config.TLS.Enabled {
		creds = credentials.NewTLS(tls.ToConfig())
	} else {
		creds = insecure.NewCredentials()
	}

	opts := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: s.config.Timeout,
		}),
		grpc.WithBlock(),
		grpc.WithTransportCredentials(creds),
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	s.logger.Debugf("trying to connect to %s...", s.config.Server)

	conn, err := grpc.DialContext(ctx, s.config.Server, opts...)
	if err != nil {
		return nil, fmt.Errorf("shipper connection failed with: %w", err)
	}
	s.logger.Debugf("connect to %s established.", s.config.Server)
	s.conn = conn
	s.client = sc.NewProducerClient(conn)

	return s, nil
}

func (s shipper) Publish(e publisher.Event) {
	msg := messages.PublishRequest{
		Events: []*messages.Event{e.ShipperMessage},
	}

	r, err := s.client.PublishEvents(context.TODO(), &msg, grpc.EmptyCallOption{})
	if status.Code(err) != codes.OK || r == nil {
		s.logger.Errorf("failed to publish event: %s", err)
		return
	}

	s.logger.Debugf(
		"shipper reply: UUID: %s, AcceptedCount: %d, AcceptedIndex: %d, PersistedIndex: %d",
		r.GetUuid(),
		r.GetAcceptedCount(),
		r.GetAcceptedIndex(),
		r.GetPersistedIndex(),
	)

	s.logger.Debug("published event")
}

func (s shipper) PublishAll(events []publisher.Event) {
	msg := messages.PublishRequest{
		Events: make([]*messages.Event, 0, len(events)),
	}

	for _, evt := range events {
		msg.Events = append(msg.Events, evt.ShipperMessage)
	}

	r, err := s.client.PublishEvents(context.TODO(), &msg, nil)
	if status.Code(err) != codes.OK || r == nil {
		s.logger.Errorf("failed to publish a batch of %d events: %s", len(events), err)
		return
	}

	s.logger.Debugf(
		"shipper reply: UUID: %s, AcceptedCount: %d, AcceptedIndex: %d, PersistedIndex: %d",
		r.GetUuid(),
		r.GetAcceptedCount(),
		r.GetAcceptedIndex(),
		r.GetPersistedIndex(),
	)

	s.logger.Debugf("published %d events", len(events))
}

func (s shipper) Close() error {
	s.logger.Debug("shutdown successful")
	return nil
}

// func (s *shipper) Connect() error {
// 	tls, err := tlscommon.LoadTLSConfig(s.config.TLS)
// 	if err != nil {
// 		return fmt.Errorf("invalid shipper TLS configuration: %w", err)
// 	}

// 	var creds credentials.TransportCredentials
// 	if s.config.TLS != nil && s.config.TLS.Enabled != nil && *s.config.TLS.Enabled {
// 		creds = credentials.NewTLS(tls.ToConfig())
// 	} else {
// 		creds = insecure.NewCredentials()
// 	}

// 	opts := []grpc.DialOption{
// 		grpc.WithConnectParams(grpc.ConnectParams{
// 			MinConnectTimeout: s.config.Timeout,
// 		}),
// 		grpc.WithBlock(),
// 		grpc.WithTransportCredentials(creds),
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
// 	defer cancel()

// 	s.logger.Debugf("trying to connect to %s...", s.config.Server)

// 	conn, err := grpc.DialContext(ctx, s.config.Server, opts...)
// 	if err != nil {
// 		return fmt.Errorf("shipper connection failed with: %w", err)
// 	}
// 	s.logger.Debugf("connect to %s established.", s.config.Server)
// 	s.conn = conn
// 	s.client = sc.NewProducerClient(conn)

// 	return nil
// }
