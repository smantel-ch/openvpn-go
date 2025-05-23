package openvpn

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

func (vc *VPNClient) waitForConnection(ctx context.Context, result chan error) {
	for {
		select {
		case line := <-vc.logs:
			vc.logsBuffer = append(vc.logsBuffer, line)
			if strings.Contains(line, "Initialization Sequence Completed") {
				vc.sendStatus(StatusConnected)
				result <- nil
				return
			}
		case err := <-vc.errors:
			vc.sendStatus(StatusError)
			result <- err
			return
		case <-ctx.Done():
			_ = vc.Disconnect()
			result <- ErrTimeout
			return
		}
	}
}

func (vc *VPNClient) prepareFiles() error {
	configFile, err := os.CreateTemp("", "*.ovpn")
	if err != nil {
		return err
	}
	if _, err := configFile.Write(vc.config); err != nil {
		configFile.Close()
		return err
	}
	configFile.Close()
	vc.configPath = configFile.Name()

	authFile, err := os.CreateTemp("", "*.auth")
	if err != nil {
		os.Remove(vc.configPath)
		return err
	}
	if _, err := authFile.WriteString(vc.username + "\n" + vc.password + "\n"); err != nil {
		authFile.Close()
		os.Remove(vc.configPath)
		os.Remove(authFile.Name())
		return err
	}
	authFile.Close()
	vc.authPath = authFile.Name()

	return nil
}

func (vc *VPNClient) pipeOutput(r io.ReadCloser) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		vc.sendLog(line)
		if strings.Contains(line, "Initialization Sequence Completed") {
			vc.sendStatus(StatusConnected)
		}
	}
}

func (vc *VPNClient) monitor() {
	err := vc.cmd.Wait()

	vc.processLock.Lock()
	defer vc.processLock.Unlock()

	vc.running = false
	vc.cleanup()

	if err != nil {
		vc.sendStatus(StatusError)
		vc.sendError(fmt.Errorf("%w: %v", ErrConnectionFailed, err))
	} else {
		vc.sendStatus(StatusDisconnected)
	}
}

func (vc *VPNClient) cleanup() {
	vc.cleanupTempFiles()
	if !vc.closed {
		close(vc.logs)
		close(vc.status)
		close(vc.errors)
		vc.closed = true
	}
}

func (vc *VPNClient) cleanupTempFiles() {
	if vc.configPath != "" {
		_ = os.Remove(vc.configPath)
		vc.configPath = ""
	}
	if vc.authPath != "" {
		_ = os.Remove(vc.authPath)
		vc.authPath = ""
	}
}

func (vc *VPNClient) sendLog(log string) {
	vc.logsBuffer = append(vc.logsBuffer, log)
	select {
	case vc.logs <- log:
	default:
	}
}

func (vc *VPNClient) sendStatus(s VPNStatus) {
	vc.currentStatus = s
	select {
	case vc.status <- s:
	default:
	}
}

func (vc *VPNClient) sendError(err error) {
	select {
	case vc.errors <- err:
	default:
	}
}
