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
				result <- ErrConnectionFailed
				return
			}
		case err := <-vc.errors:
			result <- err
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

		if logger != nil {
			logger.Debugf(line)
		}

		if err := detectLogError(line); err != nil {
			vc.lastErrorLine = err.Error()
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
	defer func() { _ = recover() }()
	select {
	case vc.errors <- err:
	default:
	}
}

func (vc *VPNClient) forceKillIfStillRunning() error {
	if vc.cmd != nil && vc.cmd.Process != nil {
		return vc.cmd.Process.Kill()
	}
	return nil
}

func killLingeringOpenVPN() ([]int, error) {
	out, err := exec.Command("pgrep", "openvpn").Output()
	if err != nil {
		return nil, nil // No lingering processes found
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var pids []int
	for _, line := range lines {
		pids = append(pids, parsePID(line))
	}
	err = exec.Command("pkill", "-f", "openvpn").Run()
	return pids, err
}

func parsePID(pidStr string) int {
	var pid int
	fmt.Sscanf(pidStr, "%d", &pid)
	return pid
}
