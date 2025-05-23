package openvpn

import (
	"testing"
	"time"
)

func TestStatusLifecycle(t *testing.T) {
	client := &VPNClient{
		status: make(chan VPNStatus, 5),
		errors: make(chan error, 5),
		logs:   make(chan string, 5),
	}

	client.sendStatus(StatusInitializing)
	client.sendStatus(StatusConnected)
	client.sendStatus(StatusDisconnected)

	statuses := []VPNStatus{}
collect:
	for {
		select {
		case s := <-client.StatusChan():
			statuses = append(statuses, s)
		case <-time.After(10 * time.Millisecond):
			break collect
		}
	}

	if len(statuses) != 3 {
		t.Errorf("expected 3 status updates, got %d", len(statuses))
	}
	if statuses[1] != StatusConnected {
		t.Errorf("expected StatusConnected as second status, got %s", statuses[1])
	}
}
