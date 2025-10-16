package service

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
)

const (
	appName = "tbot"
)

var (
	svcName = fmt.Sprintf("%s.service", appName)
)

func Install(configPath string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("detected OS of %s - service is only supported on Linux", runtime.GOOS)
	}

	if configPath == "" {
		return errors.New("config path required")
	}

	destPath := filepath.Join("/usr/local/bin", appName)
	executedPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting binary path: %w", err)
	}

	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	username := u.Username

	if err := ensureBinary(executedPath, destPath); err != nil {
		return err
	}

	if err := installService(username, appName, configPath); err != nil {
		return fmt.Errorf("installing service: %w", err)
	}

	return nil
}

func ShowLogs() error {
	journalCmd := exec.Command("journalctl", "-u", appName, "-f")
	journalCmd.Stdout = os.Stdout
	journalCmd.Stderr = os.Stderr
	journalCmd.Stdin = os.Stdin

	if err := journalCmd.Run(); err != nil {
		return fmt.Errorf("failed to show service logs: %w", err)
	}

	return nil
}

func ensureBinary(src, dst string) error {
	_, err := os.Stat(dst)
	if err == nil {
		return nil // already exists
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("checking if binary is in /usr/local/bin: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source binary: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating new binary: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying binary to %s: %w", dst, err)
	}

	if err := os.Chmod(dst, 0755); err != nil {
		return fmt.Errorf("granting proper permissions to binary: %w", err)
	}

	return nil
}

func installService(username, appName, configPath string) error {
	path := fmt.Sprintf("/etc/systemd/system/%s.service", appName)
	if err := os.WriteFile(path, []byte(service(username, configPath)), 0644); err != nil {
		return err
	}

	cmds := []*exec.Cmd{
		DaemonReload(),
		Enable(),
		Restart(),
	}

	for _, cmd := range cmds {
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	fmt.Println("Service installed and started.")
	return nil
}

func service(username, configPath string) string {
	return fmt.Sprintf(`
[Unit]
Description=TicketBot Server
After=network.target

[Service]
ExecStart=/usr/local/bin/tbot run --config %s
Restart=always
RestartSec=5
User=%s

[Install]
WantedBy=multi-user.target`, configPath, username)
}
