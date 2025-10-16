package service

import "os/exec"

func DaemonReload() *exec.Cmd {
	return exec.Command("systemctl", "daemon-reload")
}

func Restart() *exec.Cmd {
	return exec.Command("systemctl", "restart", svcName)
}

func Enable() *exec.Cmd {
	return exec.Command("systemctl", "enable", svcName)
}

func Disable() *exec.Cmd {
	return exec.Command("systemctl", "disable", svcName)
}

func Start() *exec.Cmd {
	return exec.Command("systemctl", "start", svcName)
}

func Stop() *exec.Cmd {
	return exec.Command("systemctl", "stop", svcName)
}
