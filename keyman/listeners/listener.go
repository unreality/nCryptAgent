package listeners

import (
	"context"
	"golang.org/x/crypto/ssh/agent"
)

const (
	STATUS_OK      = "ok"
	STATUS_STOPPED = "stopped"

	ERR_DISABLE = 0
	ERR_ABORTED = 1
)

type Listener interface {
	Run(ctx context.Context, agent agent.Agent) error
	Name() string
	Stop() error
	LastError() error
	Running() bool
}

type ListenerError struct {
	msg  string // description of error
	code int
}

func (e *ListenerError) Error() string { return e.msg }
func (e *ListenerError) Code() int     { return e.code }
