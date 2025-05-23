package openvpn

import (
	"context"
	"os/exec"
	"sync"
	"time"
)

type VPNClient struct {
	config   []byte
	username string
	password string

	cmd         *exec.Cmd
	cancel      context.CancelFunc
	running     bool
	processLock sync.Mutex

	configPath string
	authPath   string

	logs   chan string
	status chan VPNStatus
	errors chan error

	currentStatus VPNStatus
	logsBuffer    []string
	closed        bool
}

func NewVPNClient(config []byte, username, password string) (*VPNClient, error) {
	if _, err := exec.LookPath("openvpn"); err != nil {
		return nil, ErrBinaryNotFound
	}

	return &VPNClient{
		config:   config,
		username: username,
		password: password,
		logs:     make(chan string, 100),
		status:   make(chan VPNStatus, 10),
		errors:   make(chan error, 2),
	}, nil
}

func (vc *VPNClient) Connect(ctx context.Context) error {
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

	cmd := exec.CommandContext(ctx, "openvpn", "--config", vc.configPath, "--auth-user-pass", vc.authPath)
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
	vc.cleanup()
	return nil
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
