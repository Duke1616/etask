//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package language

import (
	"os/exec"
	"time"
)

// ConfigureCancellation 为不支持进程组控制的平台设置等待超时。
func ConfigureCancellation(command *exec.Cmd) *exec.Cmd {
	command.WaitDelay = 5 * time.Second
	return command
}
