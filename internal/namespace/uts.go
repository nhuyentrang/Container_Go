package namespace

import "syscall"

// SetHostname sets the UTS hostname
func SetHostname(name string) error {
	return syscall.Sethostname([]byte(name))
}
