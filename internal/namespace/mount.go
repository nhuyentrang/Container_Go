package namespace

import (
	"fmt"
	"syscall"
)

// Chroot changes root to the given path and chdirs to "/"
func Chroot(path string) error {
	// attempt chroot, then chdir
	if err := syscall.Chroot(path); err != nil {
		return fmt.Errorf("chroot %s: %w", path, err)
	}
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir after chroot: %w", err)
	}
	return nil
}
