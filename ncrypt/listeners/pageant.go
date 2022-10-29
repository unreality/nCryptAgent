package listeners

import (
	"context"
	"fmt"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"ncryptagent/ncrypt/listeners/pageant"
	"os"
	"sync"
)

type Pageant struct{}

func (p *Pageant) Name() string {
	return "Pageant/PuTTY"
}

func (p *Pageant) Status() string {
	//TODO implement me
	panic("implement me")
}

func (p *Pageant) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (p *Pageant) Start() error {
	//TODO implement me
	panic("implement me")
}

func (p *Pageant) Restart() error {
	//TODO implement me
	panic("implement me")
}

func (*Pageant) Run(ctx context.Context, sshagent agent.Agent) error {
	debug := true
	if os.Getenv("WCSA_DEBUG") == "1" {
		debug = true
	}
	win, err := pageant.NewPageant(debug)
	if err != nil {
		return err
	}
	defer win.Close()

	wg := new(sync.WaitGroup)
	for {
		conn, err := win.AcceptCtx(ctx)
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
