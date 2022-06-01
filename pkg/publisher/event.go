package publisher

import "github.com/elastic/elastic-agent-libs/mapstr"

type Event struct {
	Fields  mapstr.M
	Private interface{}
}
