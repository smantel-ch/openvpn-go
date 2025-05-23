package openvpn

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func (vc *VPNClient) waitForConnection(ctx context.Context, result chan error) {
	for {
		select {
		case s := <-vc.status:
			if s == StatusConnected {
				result <- nil
				return
			} else if s == StatusError || s == StatusDisconnected {
				if vc.lastErrorLine != "" {
					result <- VPNError{
						Code:    ErrConnectionFailed,
						Message: vc.lastErrorLine,
					}
				} else {
					result <- VPNError{
						Code:    ErrConnectionFailed,
						Message: "VPN disconnected unexpectedly",
					}
				}
				return
			}

		case err := <-vc.errors:
			if vc.lastErrorLine != "" {
				result <- VPNError{
					Code:    ErrConnectionFailed,
					Message: vc.lastErrorLine,
				}
			} else {
				result <- VPNError{
					Code:    ErrConnectionFailed,
					Message: fmt.Sprintf("OpenVPN exited with error: %v", err),
				}
			}
			return

		case <-ctx.Done():
			_ = vc.Disconnect()
			_ = vc.forceKillIfStillRunning()
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

		if msg := detectLogError(line); msg != "" {
			vc.lastErrorLine = msg
		}

		if strings.Contains(line, "Initialization Sequence Completed") {
			vc.sendStatus(StatusConnected)
		}
	}
}

func (vc *VPNClient) monitor() {
	err := vc.cmd.Wait()

	vc.processLock.Lock()
	defer vc.processLock.Unlock()

	if err != nil {
		vc.sendStatus(StatusError)

		if vc.lastErrorLine != "" {
			vc.sendError(fmt.Errorf("%w: %s", ErrConnectionFailed, vc.lastErrorLine))
		} else {
			vc.sendError(fmt.Errorf("%w: %v", ErrConnectionFailed, err))
		}
	} else {
		vc.sendStatus(StatusDisconnected)
	}

	vc.running = false
	vc.cleanup()
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
	defer func() { _ = recover() }()

	select {
	case vc.logs <- log:
	default:
	}
}

func (vc *VPNClient) sendStatus(s VPNStatus) {
	vc.currentStatus = s
	defer func() { _ = recover() }()

	select {
	case vc.status <- s:
	default:
	}
}

func (vc *VPNClient) sendError(err error) {
	if err == nil {
		return
	}
	defer func() { _ = recover() }()

	select {
	case vc.errors <- err:
	default:
	}
}

func (vc *VPNClient) forceKillIfStillRunning() error {
	vc.processLock.Lock()
	defer vc.processLock.Unlock()

	if vc.cmd != nil && vc.running {
		_ = vc.cmd.Process.Kill()
		vc.running = false
	}
	return nil
}

func findOpenVPNPIDs() ([]string, error) {
	out, err := exec.Command("pgrep", "-f", "openvpn").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines, nil
}

func killLingeringOpenVPN() ([]string, error) {
	pids, err := findOpenVPNPIDs()
	if err != nil || len(pids) == 0 {
		return nil, nil // Nothing running
	}

	// Try to kill them
	cmd := exec.Command("pkill", "-f", "openvpn")
	if err := cmd.Run(); err != nil {
		return pids, err
	}
	return pids, nil
}
