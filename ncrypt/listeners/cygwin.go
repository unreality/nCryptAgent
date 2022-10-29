package listeners

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

const CYGWIN_SOCK = "ncryptagent.sock"
const TYPE_CYGWIN = "CYGWIN"

type Cygwin struct {
	running  bool
	Sockfile string
}

func (s *Cygwin) Running() bool {
	return s.running
}

func (s *Cygwin) LastError() error {
	//TODO implement me
	panic("implement me")
}

func (s *Cygwin) Name() string {
	return "cygwin/msys/GIT for windows"
}

func (s *Cygwin) Status() string {
	//TODO implement me
	panic("implement me")
}

func (s *Cygwin) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (s *Cygwin) Start() error {
	//TODO implement me
	panic("implement me")
}

func (s *Cygwin) Restart() error {
	//TODO implement me
	panic("implement me")
}

func SetListenerDeadline(l net.Listener, t time.Time) error {
	switch v := l.(type) {
	case *net.TCPListener:
		return v.SetDeadline(t)
	case *net.UnixListener:
		return v.SetDeadline(t)
	}
	return nil
}

func SetFileAttributes(path string, attr uint32) error {
	cpath, cpathErr := syscall.UTF16PtrFromString(path)
	if cpathErr != nil {
		return cpathErr
	}
	return syscall.SetFileAttributes(cpath, attr)
}

func UUIDToString(uuid [16]byte) string {
	var buf [35]byte
	dst := buf[:]
	for i := 0; i < 4; i++ {
		b := uuid[i*4 : i*4+4]
		hex.Encode(dst[i*9:i*9+8], []byte{b[3], b[2], b[1], b[0]})
		if i != 3 {
			dst[9*i+8] = '-'
		}
	}
	return string(buf[:])
}

func createCygwinSocket(filename string, port int) ([]byte, error) {
	os.Remove(filename)
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}
	var uuid [16]byte
	_, err = rand.Read(uuid[:])
	if err != nil {
		return nil, err
	}
	file.WriteString(fmt.Sprintf("!<socket >%d s %s", port, UUIDToString(uuid)))
	file.Close()
	if err := SetFileAttributes(filename, syscall.FILE_ATTRIBUTE_SYSTEM|syscall.FILE_ATTRIBUTE_READONLY); err != nil {
		return nil, err
	}
	return uuid[:], nil
}

func cygwinHandshake(conn net.Conn, uuid []byte) error {
	var cuuid [16]byte
	_, err := conn.Read(cuuid[:])
	if err != nil {
		return err
	}
	if !bytes.Equal(uuid[:], cuuid[:]) {
		return fmt.Errorf("invalid uuid")
	}
	conn.Write(uuid[:])
	pidsUids := make([]byte, 12)
	_, err = conn.Read(pidsUids[:])
	if err != nil {
		return err
	}
	pid := os.Getpid()
	gid := pid // for cygwin's AF_UNIX -> AF_INET, pid = gid
	binary.LittleEndian.PutUint32(pidsUids, uint32(pid))
	binary.LittleEndian.PutUint32(pidsUids[8:], uint32(gid))
	if _, err = conn.Write(pidsUids); err != nil {
		return err
	}
	return nil
}

func (s *Cygwin) Run(ctx context.Context, sshagent agent.Agent) error {
	//home, err := os.UserConfigDir()
	//if err != nil {
	//	return err
	//}
	//Sockfile := filepath.Join(home, CYGWIN_SOCK)
	//s.Sockfile = Sockfile
	//fmt.Printf("CYGWIN socket at: %s\n", s.Sockfile)

	// listen tcp socket
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	defer func() {
		defer l.Close()
		os.Remove(s.Sockfile)
	}()
	// cygwin socket uuid
	port := l.Addr().(*net.TCPAddr).Port
	uuid, err := createCygwinSocket(s.Sockfile, port)
	if err != nil {
		return err
	}
	s.running = true
	// loop
	wg := new(sync.WaitGroup)
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		default:
		}
		SetListenerDeadline(l, time.Now().Add(time.Second))
		conn, err := l.Accept()
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			continue
		}
		if err != nil {
			return err
		}
		err = cygwinHandshake(conn, uuid)
		if err != nil {
			conn.Close()
			continue
		}
		wg.Add(1)
		go func() {
			err := agent.ServeAgent(sshagent, conn)
			if err != nil && err != io.EOF {
				fmt.Println(err.Error())
			}
			wg.Done()
		}()
	}
}
