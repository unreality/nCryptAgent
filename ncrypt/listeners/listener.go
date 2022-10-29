package listeners

import (
	"context"
	"golang.org/x/crypto/ssh/agent"
)

const (
	STATUS_OK      = "ok"
	STATUS_STOPPED = "stopped"
)

type Listener interface {
	Run(ctx context.Context, agent agent.Agent) error
	Name() string
	Stop() error
	LastError() error
	Running() bool
}
