package openvpn

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestNewVPNClient_BinaryNotFound(t *testing.T) {
	// Temporarily remove PATH to simulate missing binary
	originalPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "")
	defer os.Setenv("PATH", originalPath)

	_, err := NewVPNClient([]byte("dummy"), "user", "pass")
	if !errors.Is(err, ErrBinaryNotFound) {
		t.Errorf("expected ErrBinaryNotFound, got %v", err)
	}
}

func TestVPNClient_AlreadyRunning(t *testing.T) {
	if _, err := exec.LookPath("openvpn"); err != nil {
		t.Skip("openvpn not installed")
	}

	config := []byte("client\ndev null\nproto udp\nremote 127.0.0.1 1194\nresolv-retry infinite\nnobind\npersist-key\npersist-tun\nverb 3")
	client, err := NewVPNClient(config, "user", "pass")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go client.Connect(ctx)
	time.Sleep(100 * time.Millisecond)

	if err := client.Connect(ctx); !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("expected ErrAlreadyRunning, got %v", err)
	}

	_ = client.Disconnect()
}

func TestVPNClient_StatusInitial(t *testing.T) {
	client := &VPNClient{
		logs:          make(chan string, 10),
		status:        make(chan VPNStatus, 10),
		errors:        make(chan error, 1),
		currentStatus: StatusInitializing,
	}
	if client.Status() != StatusInitializing {
		t.Errorf("expected StatusInitializing, got %v", client.Status())
	}
}
