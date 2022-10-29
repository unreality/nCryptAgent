package listeners

import (
	"context"
	"fmt"
	"github.com/Microsoft/go-winio"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"sync"
)

const NAMED_PIPE = "\\\\.\\pipe\\openssh-ssh-agent"

type NamedPipe struct {
	running bool
}

func (s *NamedPipe) Name() string {
	return "Named Pipe"
}

func (s *NamedPipe) Status() string {
	//TODO implement me
	panic("implement me")
}

func (s *NamedPipe) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (s *NamedPipe) Start() error {
	//TODO implement me
	panic("implement me")
}

func (s *NamedPipe) Restart() error {
	//TODO implement me
	panic("implement me")
}

func (s *NamedPipe) Run(ctx context.Context, sshagent agent.Agent) error {
	var cfg = &winio.PipeConfig{}
	pipe, err := winio.ListenPipe(NAMED_PIPE, cfg)
	if err != nil {
		return err
	}

	s.running = true
	defer pipe.Close()

	wg := new(sync.WaitGroup)
	// context cancelled
	go func() {
		<-ctx.Done()
		wg.Wait()
	}()
	// loop
	for {
		conn, err := pipe.Accept()
		fmt.Println("Got an agent connection")
		if err != nil {
			if err != winio.ErrPipeListenerClosed {
				return err
			}
			return nil
		}
		wg.Add(1)
		go func() {
			err := agent.ServeAgent(sshagent, conn)
			if err != nil && err != io.EOF {
				println(err.Error())
			}
			wg.Done()
		}()
	}
}
