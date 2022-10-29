package listeners

import (
	"context"
	"golang.org/x/crypto/ssh/agent"
)

const (
	STATUS_OK = "ok"
)

type Listener interface {
	Run(ctx context.Context, agent agent.Agent) error
	Name() string
	Status() string
	Stop() error
	Start() error
	Restart() error
}
