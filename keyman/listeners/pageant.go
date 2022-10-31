package listeners

import (
	"context"
	"fmt"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"ncryptagent/keyman/listeners/pageant"
	"os"
	"sync"
)

const TYPE_PAGEANT = "PAGEANT"

type Pageant struct {
	running   bool
	win       *pageant.PageantWindow
	lastError error
}

func (p *Pageant) Running() bool {
	return p.running
}

func (p *Pageant) LastError() error {
	return p.lastError
}

func (p *Pageant) Name() string {
	return "Pageant/PuTTY"
}

func (p *Pageant) Status() string {
	//TODO implement me
	panic("implement me")
}

func (p *Pageant) Stop() error {
	p.win.Close()
	return nil
}

func (p *Pageant) Start() error {
	//TODO implement me
	panic("implement me")
}

func (p *Pageant) Restart() error {
	//TODO implement me
	panic("implement me")
}

func (p *Pageant) Run(ctx context.Context, sshagent agent.Agent) error {
	debug := true
	var err error
	if os.Getenv("WCSA_DEBUG") == "1" {
		debug = true
	}
	p.win, err = pageant.NewPageant(debug)
	if err != nil {
		return err
	}
	p.running = true
	defer func() { p.running = false }()
	defer p.win.Close()

	wg := new(sync.WaitGroup)
	for {
		conn, err := p.win.AcceptCtx(ctx)
		fmt.Println("Got pageant connection")
		if err != nil {
			if err != io.ErrClosedPipe {
				return err
			}
			return nil
		}
		wg.Add(1)
		go func() {
			fmt.Println("Handling agent connection")
			defer conn.Close()
			err := agent.ServeAgent(sshagent, conn)
			if err != nil && err != io.EOF {
				fmt.Println(err.Error())
			}
			wg.Done()
		}()
	}
}
