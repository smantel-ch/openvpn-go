// Suggestion: Add a brief package comment explaining what this module does.

package openvpn

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

type VPNClient struct {
	config        []byte
	username      string
	password      string
	cmd           *exec.Cmd
	cancel        context.CancelFunc
	running       bool
	processLock   sync.Mutex
	configPath    string
	authPath      string
	logs          chan string
	status        chan VPNStatus
	errors        chan error
	currentStatus VPNStatus
	logsBuffer    []string
	closed        bool
	lastErrorLine string
}

func NewVPNClient() (*VPNClient, error) {
	if _, err := exec.LookPath("openvpn"); err != nil {
		return nil, ErrBinaryNotFound
	}
	return &VPNClient{
		logs:   make(chan string, 100),
		status: make(chan VPNStatus, 10),
		errors: make(chan error, 2),
	}, nil
}

func (vc *VPNClient) SetCredentials(username, password string) {
	vc.username = username
	vc.password = password
}

func (vc *VPNClient) SetConfig(config []byte) {
	vc.config = config
}

func (vc *VPNClient) Connect(ctx context.Context) error {

	if pids, err := killLingeringOpenVPN(); err != nil {
		return fmt.Errorf("%w: could not terminate OpenVPN process(es): %v — %v", ErrZombieProcess, pids, err)
	}

	vc.processLock.Lock()
	defer vc.processLock.Unlock()

	if vc.running {
		return ErrAlreadyRunning
	}

	if err := vc.prepareFiles(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	vc.cancel = cancel

	cmd := exec.CommandContext(ctx, "openvpn", "--config", vc.configPath, "--auth-user-pass", vc.authPath, "--auth-nocache")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	vc.cmd = cmd
	vc.running = true
	vc.sendStatus(StatusInitializing)

	go vc.pipeOutput(stdout)
	go vc.pipeOutput(stderr)
	go vc.monitor()

	result := make(chan error, 1)
	go vc.waitForConnection(ctx, result)

	return <-result
}

func (vc *VPNClient) Disconnect() error {
	vc.processLock.Lock()
	defer vc.processLock.Unlock()

	if !vc.running {
		return nil
	}

	vc.cancel()
	vc.running = false
	vc.sendStatus(StatusDisconnected)
	vc.cleanup()
	return nil
}

func (vc *VPNClient) DisconnectAndWait(ctx context.Context) error {
	if err := vc.Disconnect(); err != nil {
		return err
	}
	for {
		select {
		case s, ok := <-vc.StatusChan():
			if !ok {
				if vc.Status() == StatusDisconnected || vc.Status() == StatusError {
					return nil
				}
				return ErrDisconnectFailed
			}
			if s == StatusDisconnected || s == StatusError {
				return nil
			}
		case err := <-vc.ErrorsChan():
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (vc *VPNClient) Reconnect(ctx context.Context) error {
	if err := vc.Disconnect(); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return vc.Connect(ctx)
}

func (vc *VPNClient) LogsChan() <-chan string      { return vc.logs }
func (vc *VPNClient) StatusChan() <-chan VPNStatus { return vc.status }
func (vc *VPNClient) ErrorsChan() <-chan error     { return vc.errors }
func (vc *VPNClient) Status() VPNStatus            { return vc.currentStatus }
