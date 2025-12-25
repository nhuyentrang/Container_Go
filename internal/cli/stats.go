package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	memBase  = "/sys/fs/cgroup/memory/containergo"
	pidsBase = "/sys/fs/cgroup/pids/containergo"
	cpuBase  = "/sys/fs/cgroup/cpu/containergo"
)

func Stats(watch bool) {
	printHeader := func() {
		fmt.Printf("%-16s %-10s %-10s %-5s\n",
			"CONTAINER", "CPU(ms)", "MEM(MB)", "PIDS")
	}

	for {
		fmt.Print("\033[H\033[2J") // clear terminal
		printHeader()

		entries, err := os.ReadDir(memBase)
		if err != nil {
			fmt.Println("cannot read", memBase)
			return
		}

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}

			id := e.Name()

			mem := readInt(filepath.Join(memBase, id, "memory.usage_in_bytes")) / 1024 / 1024
			pids := readInt(filepath.Join(pidsBase, id, "pids.current"))
			cpu := readInt(filepath.Join(cpuBase, id, "cpuacct.usage")) / 1_000_000

			fmt.Printf("%-16s %-10d %-10d %-5d\n",
				id, cpu, mem, pids)
		}

		if !watch {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func readInt(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	v, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	return int(v)
}
