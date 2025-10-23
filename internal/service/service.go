package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	if _, err := os.Stat(dst); err == nil {
		slog.Debug("destination binary already exists, checking hash", "path", dst)
		match, err := filesEqual(src, dst)
		if err != nil {
			return fmt.Errorf("comparing binaries: %w", err)
		}

		if match {
			slog.Debug("file hashes match, no action needed")
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking if binary is in /usr/local/bin: %w", err)
	}

	dir := filepath.Dir(dst)
	tmp := filepath.Join(dir, filepath.Base(dst)+".tmp")

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source binary: %w", err)
	}
	slog.Debug("opened source binary", "path", src)
	defer in.Close()

	out, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("creating temporary binary: %w", err)
	}
	slog.Debug("created destination binary", "path", dst)

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return fmt.Errorf("copying binary to %s: %w", dst, err)
	}
	slog.Debug("copied binary", "src", src, "dst", dst)

	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("closing temp binary: %w", err)
	}

	perm := os.FileMode(0755)
	if err := os.Chmod(tmp, perm); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("chmod temp binary: %w", err)
	}
	slog.Debug("binary permissions changed", "path", dst, "permissions", perm)

	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("installing binary: %w", err)
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

func filesEqual(a, b string) (bool, error) {
	ha, err := fileHash(a)
	if err != nil {
		return false, err
	}

	hb, err := fileHash(b)
	if err != nil {
		return false, err
	}

	return ha == hb, nil
}

func fileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to copy file content to hash: %w", err)
	}

	hasbBytes := hash.Sum(nil)
	hashStr := fmt.Sprintf("%x", hasbBytes)

	return hashStr, nil
}
