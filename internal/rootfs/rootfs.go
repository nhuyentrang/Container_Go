package rootfs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// PrepareBusyboxRootfs is a tiny helper suggestion: copy busybox static into folder rootfs.
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
	bb := strings.TrimSpace(string(out)) 
	if bb == "" {
		return fmt.Errorf("busybox not found in PATH")
	}
	// copy busybox to dest/bin
	if err := exec.Command("cp", bb, dest+"/bin/busybox").Run(); err != nil {
		return err
	}
	_ = os.Symlink("/bin/busybox", dest+"/bin/sh")

	_ = os.MkdirAll(dest+"/proc", 0755)
	_ = os.MkdirAll(dest+"/dev", 0755)
	_ = os.MkdirAll(dest+"/sys", 0755)
	_ = os.MkdirAll(dest+"/tmp", 0755)

	return nil
}

// Setup switches current process root to `rootfsPath` using chroot,
// mounts /proc inside it, and returns a cleanup function.
func Setup(rootfsPath string) (func() error, error) {
	abs, err := filepath.Abs(rootfsPath)
	if err != nil {
		return nil, fmt.Errorf("abs rootfs: %w", err)
	}
	st, err := os.Stat(abs)
	if err != nil || !st.IsDir() {
		return nil, fmt.Errorf("rootfs not found or not a dir: %s", abs)
	}

	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return nil, fmt.Errorf("make / private: %w", err)
	}

	if err := syscall.Chroot(abs); err != nil {
		return nil, fmt.Errorf("chroot(%s): %w", abs, err)
	}
	if err := syscall.Chdir("/"); err != nil {
		return nil, fmt.Errorf("chdir(/): %w", err)
	}

	_ = os.MkdirAll("/proc", 0555)
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return nil, fmt.Errorf("mount proc: %w", err)
	}

	cleanup := func() error {
		_ = syscall.Unmount("/proc", 0)
		return nil
	}
	return cleanup, nil
}

func mountProc() error {
	_ = os.MkdirAll("/proc", 0555)
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}
	return nil
}