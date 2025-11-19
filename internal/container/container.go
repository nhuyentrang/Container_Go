package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"containergo/internal/namespace"
)

func Run(command []string) {
	if len(command) == 0 {
		fmt.Fprintln(os.Stderr, "no command provided")
		os.Exit(1)
	}

	fmt.Printf("Running %v as pid %d\n", command, os.Getpid())

	// re-exec chính bản thân nhưng với arg "child" để code đi nhánh child()
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, command...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// tạo namespace mới: UTS (hostname), PID, Mount
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
	}

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run error: %v\n", err)
		os.Exit(1)
	}
}

func Child(command []string) {
	if len(command) == 0 {
		fmt.Fprintln(os.Stderr, "child: no command")
		os.Exit(1)
	}

	fmt.Printf("Running child %v as pid %d\n", command, os.Getpid())

	// set hostname in UTS namespace
	if err := namespace.SetHostname("mycontainer"); err != nil {
		fmt.Fprintf(os.Stderr, "sethostname error: %v\n", err)
		// continue anyway
	}

	// chroot into "/" (placeholder). Later replace with proper rootfs/pivot_root.
	if err := namespace.Chroot("/"); err != nil {
		fmt.Fprintf(os.Stderr, "chroot error: %v\n", err)
		// not fatal for demo basic
	}

	// finally exec the requested command inside the container (PID namespace, UTS applied)
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "child run error: %v\n", err)
		os.Exit(1)
	}
}
