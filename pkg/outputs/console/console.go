package console

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
)

// ConsolePipeline implements pkg/publisher.Pipeline
type ConsolePipeline struct {
	out io.Writer
}

// Console implements pkg/publisher.Client
type Console struct {
	logger *logp.Logger
}

func NewPipeline(out io.Writer) ConsolePipeline {
	return ConsolePipeline{out: out}
}

// ConnectWith returns a Client using the given configuration
func (c ConsolePipeline) ConnectWith(publisher.ClientConfig) (publisher.Client, error) {
	return nil, errors.New("not implemented")
}

// Connect returns a client using the default configuration
func (c ConsolePipeline) Connect() (publisher.Client, error) {
	return Console{}, nil
}

// Publish publishes a single event to stdout
func (c Console) Publish(event publisher.Event) {
	os.Stdout.Write([]byte(fmt.Sprint(event.ShipperMessage.Fields)))
}

// PublishAll calls c.Publish for every event
func (c Console) PublishAll(events []publisher.Event) {
	for _, event := range events {
		c.Publish(event)
	}
}

// Close no op function, stdout does not need closing
func (c Console) Close() error {
	return nil
}
