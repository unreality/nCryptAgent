package listeners

import (
	"context"
	"golang.org/x/crypto/ssh/agent"
)

type Listener interface {
	Run(ctx context.Context, agent agent.Agent) error
}
