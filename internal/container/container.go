package container

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"containergo/internal/cgroups"
	"containergo/internal/namespace"
	"containergo/internal/rootfs"
)

const defaultRootfs = "./rootfs"

type runOptions struct {
	rootfs  string
	limits  cgroups.Limits
	memStr  string // giữ chuỗi mem để truyền xuống child (debug/inspect)
}

// parseRunArgs: parse flags cho `run`, trả về options và command còn lại.
func parseRunArgs(args []string) (runOptions, []string, error) {
	var opt runOptions

	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&opt.rootfs, "rootfs", defaultRootfs, "path to root filesystem directory")
	fs.StringVar(&opt.memStr, "mem", "", "memory limit (e.g. 100m, 1g). empty = unlimited")
	fs.IntVar(&opt.limits.CPUPercent, "cpu", 0, "cpu limit percent 1..100. 0 = unlimited")
	fs.IntVar(&opt.limits.PidsMax, "pids", 0, "max number of processes. 0 = unlimited")

	if err := fs.Parse(args); err != nil {
		return runOptions{}, nil, err
	}

	mb, err := cgroups.ParseMemory(opt.memStr)
	if err != nil {
		return runOptions{}, nil, err
	}
	opt.limits.MemoryBytes = mb

	return opt, fs.Args(), nil
}

// parseChildArgs: parse flags cho `child`, trả về options và command.
// (Child parse để đồng bộ args + in log; cgroups limit set ở parent.)
func parseChildArgs(args []string) (runOptions, []string, error) {
	var opt runOptions

	fs := flag.NewFlagSet("child", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&opt.rootfs, "rootfs", "", "path to root filesystem directory")
	fs.StringVar(&opt.memStr, "mem", "", "memory limit (debug)")
	fs.IntVar(&opt.limits.CPUPercent, "cpu", 0, "cpu limit percent (debug)")
	fs.IntVar(&opt.limits.PidsMax, "pids", 0, "max processes (debug)")

	if err := fs.Parse(args); err != nil {
		return runOptions{}, nil, err
	}

	mb, _ := cgroups.ParseMemory(opt.memStr)
	opt.limits.MemoryBytes = mb

	return opt, fs.Args(), nil
}

func Run(args []string) {
	opt, command, err := parseRunArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse run args error: %v\n", err)
		os.Exit(1)
	}
	if len(command) == 0 {
		fmt.Fprintln(os.Stderr, "usage: container run [--rootfs PATH] [--mem 100m] [--cpu 50] [--pids 64] <command...>")
		os.Exit(1)
	}

	fmt.Printf("Running %v as pid %d\n", args, os.Getpid())

	// ✅ chuẩn bị rootfs tối thiểu (busybox + /bin/sh)
	if err := rootfs.PrepareBusyboxRootfs(opt.rootfs); err != nil {
		fmt.Fprintf(os.Stderr, "prepare rootfs error: %v\n", err)
		os.Exit(1)
	}

	// ✅ Truyền nguyên args xuống child để child parse y hệt (rootfs/mem/cpu/pids + command)
	// Lưu ý: args ở đây là phần sau "run" (đúng như main.go đang gọi container.Run(os.Args[2:]))
	childArgs := append([]string{"child"}, args...)

	cmd := exec.Command("/proc/self/exe", childArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
	}

	// ✅ Start (không dùng Run) để lấy PID host của child => dùng cho cgroups
	if err := cmd.Start(); err != nil {
		fmt.Printf("Container host PID: %d\n", cmd.Process.Pid)
		fmt.Fprintf(os.Stderr, "start error: %v\n", err)
		os.Exit(1)
	}

	// ✅ tạo cgroup và apply limit vào PID host của child
	containerID := fmt.Sprintf("pid-%d", cmd.Process.Pid)
	mgr, err := cgroups.NewManager(containerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cgroup init error: %v\n", err)
		_ = cmd.Process.Kill()
		os.Exit(1)
	}
	defer func() { _ = mgr.Cleanup() }()

	if err := mgr.Apply(cmd.Process.Pid, opt.limits); err != nil {
		fmt.Fprintf(os.Stderr, "cgroup apply error: %v\n", err)
		_ = cmd.Process.Kill()
		os.Exit(1)
	}

	// chờ container kết thúc
	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "run error: %v\n", err)
		os.Exit(1)
	}
}

func Child(args []string) {
	opt, command, err := parseChildArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse child args error: %v\n", err)
		os.Exit(1)
	}
	if opt.rootfs == "" {
		opt.rootfs = defaultRootfs
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

	// rootfs isolation (chroot + mount /proc)
	cleanup, err := rootfs.Setup(opt.rootfs)
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
