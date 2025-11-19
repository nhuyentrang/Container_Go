package rootfs

import (
	"fmt"
	"os"
	"os/exec"
)

// PrepareBusyboxRootfs is a tiny helper suggestion: copy busybox static into folder rootfs.
// This is not mandatory now; keep as helper for later.
func PrepareBusyboxRootfs(dest string) error {
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		return nil // exists
	}
	if err := os.MkdirAll(dest+"/bin", 0755); err != nil {
		return err
	}
	// try to locate busybox on host and copy
	out, err := exec.Command("which", "busybox").Output()
	if err != nil {
		return fmt.Errorf("busybox not found: %w", err)
	}
	bb := string(out)
	if bb == "" {
		return fmt.Errorf("busybox not found in PATH")
	}
	// copy busybox to dest/bin
	return exec.Command("cp", bb, dest+"/bin/busybox").Run()
}
