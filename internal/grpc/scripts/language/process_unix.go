//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package language

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// ConfigureCancellation 确保上下文取消时终止脚本创建的整个进程组。
func ConfigureCancellation(command *exec.Cmd) *exec.Cmd {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	command.WaitDelay = 5 * time.Second
	return command
}
