package listeners

import (
	"context"
	"github.com/Microsoft/go-winio"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"log"
	"net"
	"sync"
)

const NAMED_PIPE = "\\\\.\\pipe\\openssh-ssh-agent"
const TYPE_NAMED_PIPE = "NAMED_PIPE"

type NamedPipe struct {
	running   bool
	pipe      net.Listener
	lastError error
}

func (s *NamedPipe) Running() bool {
	return s.running
}

func (s *NamedPipe) LastError() error {
	return s.lastError
}

func (s *NamedPipe) Name() string {
	return "Named Pipe"
}

func (s *NamedPipe) Status() string {
	if s.running {
		return STATUS_OK
	} else {
		return STATUS_STOPPED
	}
}

func (s *NamedPipe) Stop() error {
	return s.pipe.Close()
}

func (s *NamedPipe) Restart() error {
	//TODO implement me
	panic("implement me")
}

func (s *NamedPipe) Run(ctx context.Context, sshagent agent.Agent) error {
	var cfg = &winio.PipeConfig{}
	s.pipe, s.lastError = winio.ListenPipe(NAMED_PIPE, cfg)

	if s.lastError != nil {
		return s.lastError
	}

	s.running = true
	defer s.pipe.Close()
	defer func() { s.running = false }()

	wg := new(sync.WaitGroup)
	// context cancelled
	go func() {
		<-ctx.Done()
		wg.Wait()
	}()
	// loop
	for {
		var conn net.Conn
		conn, s.lastError = s.pipe.Accept()

		if s.lastError != nil {
			if s.lastError != winio.ErrPipeListenerClosed {
				return s.lastError
			}
			return nil
		}
		wg.Add(1)
		go func() {
			s.lastError = agent.ServeAgent(sshagent, conn)
			if s.lastError != nil && s.lastError != io.EOF {
				log.Println(s.lastError.Error())
			}
			wg.Done()
		}()
	}
}
