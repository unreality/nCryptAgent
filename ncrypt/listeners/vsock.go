package listeners

import (
	"context"
	"fmt"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/sys/windows/registry"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/Microsoft/go-winio/pkg/guid"
	"github.com/bi-zone/wmi"
)

var (
	vmWildCard, _     = guid.FromString("00000000-0000-0000-0000-000000000000")
	HyperVServiceGUID = winio.VsockServiceID(servicePort)
)

const (
	servicePort          = 0x22223333
	HyperVServiceRegPath = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization\GuestCommunicationServices`
)

// https://docs.microsoft.com/en-us/virtualization/hyper-v-on-windows/user-guide/make-integration-service
//$friendlyName = "WinCryptSSHAgent"
//$service = New-Item -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization\GuestCommunicationServices" -Name "22223333-facb-11e6-bd58-64006a7986d3"
//$service.SetValue("ElementName", $friendlyName)

type VSock struct {
	running bool
}

func (s *VSock) Name() string {
	return "WSL2"
}

func (s *VSock) Status() string {
	//TODO implement me
	panic("implement me")
}

func (s *VSock) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (s *VSock) Start() error {
	//TODO implement me
	panic("implement me")
}

func (s *VSock) Restart() error {
	//TODO implement me
	panic("implement me")
}

type vSockWorker struct {
	l        net.Listener
	sshagent agent.Agent
}

func newVSockWorker(vmid string, sshagent agent.Agent) (*vSockWorker, error) {
	vmidGUID, err := guid.FromString(vmid)
	if err != nil {
		return nil, err
	}
	pipe, err := winio.ListenHvsock(&winio.HvsockAddr{
		VMID:      vmidGUID,
		ServiceID: HyperVServiceGUID,
	})
	if err != nil {
		return nil, err
	}
	return &vSockWorker{
		l:        pipe,
		sshagent: sshagent,
	}, nil
}

func (s *vSockWorker) Run() {
	for {
		conn, err := s.l.Accept()
		if err != nil {
			return
		}
		go func() {
			agent.ServeAgent(s.sshagent, conn)
		}()
	}
}

func (s *vSockWorker) Close() {
	s.l.Close()
}

func vmidDiff(old, new []string) (add, del []string) {
	add = make([]string, 0)
	del = make([]string, 0)
	oldIDS := make(map[string]interface{})
	newIDS := make(map[string]interface{})
	for _, v := range old {
		oldIDS[v] = 0
	}
	for _, v := range new {
		newIDS[v] = 0
	}
	for _, v := range new {
		if _, ok := oldIDS[v]; !ok {
			add = append(add, v)
		}
	}
	for _, v := range old {
		if _, ok := newIDS[v]; !ok {
			del = append(del, v)
		}
	}
	return
}

func (s *VSock) wsl2Watcher(ctx context.Context, sshagent agent.Agent) {
	timeout := time.Second * 60
	ch := make(chan *ProcessEvent, 1)
	pn, err := NewProcessNotify("wslhost.exe", ch)
	if err != nil {
		// fallback to polling mode
		timeout = time.Second * 15
		println("ProcessNotify error:", err.Error())
	} else {
		pn.Start()
		defer pn.Stop()
	}
	lastVMIDs := make([]string, 0)
	workers := make(map[string]*vSockWorker)
	for {
		vmids := GetVMIDs()
		add, del := vmidDiff(lastVMIDs, vmids)
		for _, v := range add {
			w, err := newVSockWorker(v, sshagent)
			if err != nil {
				continue
			}
			workers[v] = w
			go w.Run()
		}
		for _, v := range del {
			w := workers[v]
			if w != nil {
				w.Close()
				delete(workers, v)
			}
		}
		lastVMIDs = vmids
		select {
		case <-ctx.Done():
			return
		case <-ch:
		case <-time.After(timeout):
		}
	}
}

func (s *VSock) Run(ctx context.Context, sshagent agent.Agent) error {

	if !CheckHvSocket() {
		return nil
	}

	if !CheckHVService() {
		return nil
	}

	pipe, err := winio.ListenHvsock(&winio.HvsockAddr{
		VMID:      vmWildCard,
		ServiceID: HyperVServiceGUID,
	})
	if err != nil {
		return err
	}

	s.running = true
	defer pipe.Close()

	go s.wsl2Watcher(ctx, sshagent)

	wg := new(sync.WaitGroup)
	// context cancelled
	go func() {
		<-ctx.Done()
		wg.Wait()
	}()
	// loop
	for {
		conn, err := pipe.Accept()
		if err != nil {
			return nil
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

//func (s *VSock) onClick() {
//	if !s.running {
//		s.checkHvService()
//		return
//	}
//
//	// socat 1.7.4 support vsock,
//	// `SOCKET-CONNECT` can be replaced with `VSOCK-CONNECT:2:0x22223333`
//	help := `export SSH_AUTH_SOCK=/tmp/wincrypt-hv.sock
//ss -lnx | grep -q $SSH_AUTH_SOCK
//if [ $? -ne 0 ]; then
//	rm -f $SSH_AUTH_SOCK
//  (setsid nohup socat UNIX-LISTEN:$SSH_AUTH_SOCK,fork SOCKET-CONNECT:40:0:x0000x33332222x02000000x00000000 >/dev/null 2>&1)
//fi`
//
//}

//func (s *VSock) checkHvService() {
//	if CheckHVService() {
//		utils.MessageBox("Error:", s.AppId().String()+" agent doesn't work!", utils.MB_ICONWARNING)
//		return
//	}
//
//	if utils.MessageBox(s.AppId().FullName()+":", s.AppId().String()+" agent is not working! Do you want to enable it?", utils.MB_OKCANCEL) == utils.IDOK {
//		if err := utils.RunMeElevatedWithArgs("-i"); err != nil {
//			utils.MessageBox("Install Service Error:", err.Error(), utils.MB_ICONERROR)
//		}
//	}
//}

//func ConnectHyperV() (net.Conn, error) {
//
//    s := winio.HvsockAddr{
//        VMID:      winio.HvsockGUIDParent(),
//        ServiceID: HyperVServiceGUID,
//    }
//
//    conn, err := winio.Dial(s)
//	if err != nil {
//		return nil, err
//	}
//
//	return conn, nil
//}

const afHvSock = 34      // AF_HYPERV
const sHvProtocolRaw = 1 // HV_PROTOCOL_RAW

func CheckHVService() bool {
	gcs, err := registry.OpenKey(registry.LOCAL_MACHINE, HyperVServiceRegPath, registry.READ)
	if err != nil {
		return false
	}
	defer gcs.Close()

	agentSrv, err := registry.OpenKey(gcs, HyperVServiceGUID.String(), registry.READ)
	if err != nil {
		return false
	}
	agentSrv.Close()
	return true
}

func GetVMIDs() []string {
	type Win32_Process struct {
		CommandLine string
	}
	var processes []Win32_Process
	q := wmi.CreateQuery(&processes, "WHERE Name='wslhost.exe'")
	err := wmi.Query(q, &processes)
	if err != nil {
		return nil
	}

	guids := make(map[string]interface{})

	for _, v := range processes {
		args := strings.Split(v.CommandLine, " ")
		for i := len(args) - 1; i >= 0; i-- {
			if strings.Contains(args[i], "{") {
				guids[args[i]] = nil
				break
			}
		}
	}

	results := make([]string, 0)
	for k := range guids {
		results = append(results, k[1:len(k)-1])
	}
	return results
}

func CheckHvSocket() bool {
	fd, err := syscall.Socket(afHvSock, syscall.SOCK_STREAM, sHvProtocolRaw)
	if err != nil {
		println(err.Error())
		return false
	}
	syscall.Close(fd)
	return true
}

const (
	PROCESS_CREATE = iota
	PROCESS_DELETE
	PROCESS_MODIFY
	PROCESS_ERROR
)

const processEventQuery = `
SELECT * FROM __InstanceOperationEvent
WITHIN 1
WHERE
TargetInstance ISA 'Win32_Process'
AND TargetInstance.Name='%s'`

type ProcessEvent struct {
	Type        int
	Error       error
	TimeStamp   uint64
	ProcessId   uint32
	Name        string
	CommandLine string
}

type wmiProcessEvent struct {
	TimeStamp uint64 `wmi:"TIME_CREATED"`
	System    struct {
		Class string
	} `wmi:"Path_"`
	Instance win32Process `wmi:"TargetInstance"`
}

type win32Process struct {
	ProcessId   uint32
	Name        string
	CommandLine string
}

type ProcessNotify struct {
	q      *wmi.NotificationQuery
	events chan wmiProcessEvent
	ch     chan<- *ProcessEvent
}

func NewProcessNotify(name string, ch chan<- *ProcessEvent) (*ProcessNotify, error) {
	events := make(chan wmiProcessEvent)
	q, err := wmi.NewNotificationQuery(events, fmt.Sprintf(processEventQuery, name))
	if err != nil {
		return nil, err
	}
	return &ProcessNotify{
		q:      q,
		events: events,
		ch:     ch,
	}, nil
}

func (s *ProcessNotify) Start() {
	done := make(chan error, 1)

	go func() {
		done <- s.q.StartNotifications()
	}()

	go s.dispatch(done)
}

func (s *ProcessNotify) Stop() {
	s.q.Stop()
}

func (s *ProcessNotify) dispatch(done chan error) {
	for {
		select {
		case ev := <-s.events:
			event := &ProcessEvent{
				TimeStamp:   ev.TimeStamp,
				ProcessId:   ev.Instance.ProcessId,
				Name:        ev.Instance.Name,
				CommandLine: ev.Instance.CommandLine,
			}
			switch ev.System.Class {
			case "__InstanceCreationEvent":
				event.Type = PROCESS_CREATE
			case "__InstanceDeletionEvent":
				event.Type = PROCESS_DELETE
			default:
				event.Type = PROCESS_MODIFY
			}
			s.ch <- event
		case err := <-done:
			event := &ProcessEvent{
				Type:  PROCESS_ERROR,
				Error: err,
			}
			s.ch <- event
			return
		}
	}

}
