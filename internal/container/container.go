package container

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"containergo/internal/namespace"
	"containergo/internal/rootfs"
)

const defaultRootfs = "./rootfs"

// parseRunArgs tách flags của `run` và phần command còn lại.
// Ví dụ: run --rootfs ./rootfs sh  => rootfsPath="./rootfs", cmd=["sh"]
func parseRunArgs(args []string) (rootfsPath string, cmd []string, err error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&rootfsPath, "rootfs", defaultRootfs, "path to root filesystem directory")
	// Parse sẽ “ăn” các flag kiểu --rootfs ./rootfs
	if err := fs.Parse(args); err != nil {
		return "", nil, err
	}

	cmd = fs.Args()
	return rootfsPath, cmd, nil
}

// parseChildArgs tách flags của `child` và command.
// Lưu ý: child nhận args từ Run truyền xuống (có thể chứa --rootfs ...).
func parseChildArgs(args []string) (rootfsPath string, cmd []string, err error) {
	fs := flag.NewFlagSet("child", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&rootfsPath, "rootfs", "", "path to root filesystem directory")
	if err := fs.Parse(args); err != nil {
		return "", nil, err
	}

	cmd = fs.Args()
	return rootfsPath, cmd, nil
}

func Run(args []string) {
	rootfsPath, command, err := parseRunArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse run args error: %v\n", err)
		os.Exit(1)
	}
	if len(command) == 0 {
		fmt.Fprintln(os.Stderr, "no command provided")
		fmt.Fprintln(os.Stderr, "usage: container run [--rootfs PATH] <command...>")
		os.Exit(1)
	}

	fmt.Printf("Running %v as pid %d\n", append([]string{"--rootfs", rootfsPath}, command...), os.Getpid())

	//  chuẩn bị rootfs tối thiểu (busybox + /bin/sh)
	// (Giải thích: để chroot vào rootfs không bị thiếu shell/dir cơ bản)
	if err := rootfs.PrepareBusyboxRootfs(rootfsPath); err != nil {
		fmt.Fprintf(os.Stderr, "prepare rootfs error: %v\n", err)
		os.Exit(1)
	}

	//  Truyền xuống child dạng:
	// child --rootfs <path> <cmd...>
	childArgs := append([]string{"child", "--rootfs", rootfsPath}, command...)

	cmd := exec.Command("/proc/self/exe", childArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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

func Child(args []string) {
	rootfsPath, command, err := parseChildArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse child args error: %v\n", err)
		os.Exit(1)
	}
	if rootfsPath == "" {
		// nếu quên truyền rootfs, fallback về default để không chết demo
		rootfsPath = defaultRootfs
	}
	if len(command) == 0 {
		fmt.Fprintln(os.Stderr, "child: no command")
		fmt.Fprintln(os.Stderr, "usage: container child --rootfs PATH <command...>")
		os.Exit(1)
	}

	fmt.Printf("Running child %v as pid %d\n", command, os.Getpid())

	// set hostname in UTS namespace
	if err := namespace.SetHostname("mycontainer"); err != nil {
		fmt.Fprintf(os.Stderr, "sethostname error: %v\n", err)
	}

	//  rootfs isolation (chroot + mount /proc)
	cleanup, err := rootfs.Setup(rootfsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rootfs setup error: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = cleanup() }()

	// exec command trong container
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "child run error: %v\n", err)
		os.Exit(1)
	}
}