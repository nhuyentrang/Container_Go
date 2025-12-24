package rootfs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// PrepareBusyboxRootfs: tạo rootfs tối thiểu với busybox + /bin/sh.
// Sửa: dù rootfs đã tồn tại vẫn đảm bảo các thư mục và symlink cần thiết.
func PrepareBusyboxRootfs(dest string) error {
	// luôn đảm bảo các thư mục tối thiểu tồn tại
	if err := os.MkdirAll(dest+"/bin", 0755); err != nil {
		return err
	}
	_ = os.MkdirAll(dest+"/proc", 0755)
	_ = os.MkdirAll(dest+"/dev", 0755)
	_ = os.MkdirAll(dest+"/sys", 0755)
	_ = os.MkdirAll(dest+"/tmp", 0755)

	// nếu đã có busybox rồi thì chỉ cần đảm bảo /bin/sh
	if _, err := os.Stat(dest + "/bin/busybox"); err == nil {
		_ = os.Remove(dest + "/bin/sh")
		_ = os.Symlink("/bin/busybox", dest+"/bin/sh")
		return nil
	}

	// tìm busybox trên host và copy vào rootfs
	out, err := exec.Command("which", "busybox").Output()
	if err != nil {
		return fmt.Errorf("busybox not found: %w", err)
	}
	bb := strings.TrimSpace(string(out))
	if bb == "" {
		return fmt.Errorf("busybox not found in PATH")
	}

	if err := exec.Command("cp", bb, dest+"/bin/busybox").Run(); err != nil {
		return err
	}

	// tạo /bin/sh -> busybox để chạy "sh" trong container
	_ = os.Remove(dest + "/bin/sh")
	_ = os.Symlink("/bin/busybox", dest+"/bin/sh")

	return nil
}

// Setup switches current process root to `rootfsPath` using chroot,
// mounts /dev (devtmpfs) + /proc inside it, and returns a cleanup function.
//
// Vì sao cần /dev?
// - Nếu không có /dev/null thì redirect/CLI test (yes > /dev/null, ...) sẽ lỗi.
// - devtmpfs cung cấp các device node phổ biến: null, zero, random...
func Setup(rootfsPath string) (func() error, error) {
	abs, err := filepath.Abs(rootfsPath)
	if err != nil {
		return nil, fmt.Errorf("abs rootfs: %w", err)
	}
	st, err := os.Stat(abs)
	if err != nil || !st.IsDir() {
		return nil, fmt.Errorf("rootfs not found or not a dir: %s", abs)
	}

	// chặn mount propagation để mount trong container không "lan" ra host
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return nil, fmt.Errorf("make / private: %w", err)
	}

	// đổi "/" của process sang rootfsPath
	if err := syscall.Chroot(abs); err != nil {
		return nil, fmt.Errorf("chroot(%s): %w", abs, err)
	}
	if err := syscall.Chdir("/"); err != nil {
		return nil, fmt.Errorf("chdir(/): %w", err)
	}

	// ✅ MOUNT /dev bằng devtmpfs để có /dev/null, /dev/zero,...
	// (chạy sudo nên sẽ mount được)
	_ = os.MkdirAll("/dev", 0755)
	if err := syscall.Mount("devtmpfs", "/dev", "devtmpfs", 0, ""); err != nil {
		return nil, fmt.Errorf("mount devtmpfs: %w", err)
	}

	// ✅ MOUNT /proc để ps hoạt động đúng trong PID namespace
	_ = os.MkdirAll("/proc", 0555)
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		_ = syscall.Unmount("/dev", 0)
		return nil, fmt.Errorf("mount proc: %w", err)
	}

	cleanup := func() error {
		// best-effort
		_ = syscall.Unmount("/proc", 0)
		_ = syscall.Unmount("/dev", 0)
		return nil
	}
	return cleanup, nil
}