package main

import (
	"os"
	"os/exec"
	"syscall"
	"fmt"
)

func main() {
	if len(os.Args) < 2 {
		panic("usage: container run <command>")
	}

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("unknown command")
	}
}

func run() {
	fmt.Printf("Running %v as pid %d\n", os.Args[2:], os.Getpid())

	command := os.Args[2:]
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, command...)...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Tạo namespace mới giống Docker (UTS, PID, Mount)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
	}

	cmd.Run()
}

func child() {
	fmt.Printf("Running child %v as pid %d\n", os.Args[2:], os.Getpid())

	// Đặt tên hostname trong container
	syscall.Sethostname([]byte("mycontainer"))

	// Chroot vào / (đơn giản)
	syscall.Chroot("/")
	syscall.Chdir("/")

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
}
