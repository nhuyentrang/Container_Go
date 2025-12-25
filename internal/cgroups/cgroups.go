package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Limits struct {
	MemoryBytes int64 // 0 = unlimited
	CPUPercent  int   // 0 = unlimited (1..100)
	PidsMax     int   // 0 = unlimited
}

type Manager struct {
	ID       string
	MemPath  string
	CpuPath  string
	PidsPath string
}

func NewManager(containerID string) (*Manager, error) {
	m := &Manager{
		ID:       containerID,
		MemPath:  filepath.Join("/sys/fs/cgroup/memory", "containergo", containerID),
		CpuPath:  filepath.Join("/sys/fs/cgroup/cpu", "containergo", containerID),
		PidsPath: filepath.Join("/sys/fs/cgroup/pids", "containergo", containerID),
	}

	for _, p := range []string{m.MemPath, m.CpuPath, m.PidsPath} {
		if err := os.MkdirAll(p, 0755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", p, err)
		}
	}
	return m, nil
}

func (m *Manager) Apply(pid int, lim Limits) error {
	// MEMORY
	if lim.MemoryBytes > 0 {
		if err := writeFile(filepath.Join(m.MemPath, "memory.limit_in_bytes"), strconv.FormatInt(lim.MemoryBytes, 10)); err != nil {
			return fmt.Errorf("set memory.limit_in_bytes: %w", err)
		}
	}

	// PIDS
	if lim.PidsMax > 0 {
		if err := writeFile(filepath.Join(m.PidsPath, "pids.max"), strconv.Itoa(lim.PidsMax)); err != nil {
			return fmt.Errorf("set pids.max: %w", err)
		}
	}

	// CPU (CFS quota)
	if lim.CPUPercent > 0 {
		if lim.CPUPercent < 1 || lim.CPUPercent > 100 {
			return fmt.Errorf("cpu percent must be 1..100")
		}
		period := 100000 // 100ms
		quota := period * lim.CPUPercent / 100
		if err := writeFile(filepath.Join(m.CpuPath, "cpu.cfs_period_us"), strconv.Itoa(period)); err != nil {
			return fmt.Errorf("set cpu.cfs_period_us: %w", err)
		}
		if err := writeFile(filepath.Join(m.CpuPath, "cpu.cfs_quota_us"), strconv.Itoa(quota)); err != nil {
			return fmt.Errorf("set cpu.cfs_quota_us: %w", err)
		}
	}

	// Add PID into cgroups (v1): write to both "tasks" and "cgroup.procs".
	// On some hybrid setups, one of them may be ignored; writing both is robust.
	pidStr := strconv.Itoa(pid)

	add := func(dir string) error {
		// ưu tiên cgroup.procs trước (process-based)
		if err := writeFile(filepath.Join(dir, "cgroup.procs"), pidStr); err != nil {
			// nếu file không tồn tại thì thử tasks, còn nếu tồn tại mà lỗi thì báo
			if !os.IsNotExist(err) {
				return err
			}
		}
		if err := writeFile(filepath.Join(dir, "tasks"), pidStr); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	}

	// add vào từng controller
	if err := add(m.MemPath); err != nil {
		return fmt.Errorf("add pid to memory cgroup: %w", err)
	}
	if err := add(m.CpuPath); err != nil {
		return fmt.Errorf("add pid to cpu cgroup: %w", err)
	}
	if err := add(m.PidsPath); err != nil {
		return fmt.Errorf("add pid to pids cgroup: %w", err)
	}
	return nil
}

func writeFile(path, val string) error {
	return os.WriteFile(path, []byte(val), 0644)
}

// ParseMemory supports "100m", "1g", "500k" or bytes.
func ParseMemory(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, nil
	}
	if isDigits(s) {
		return strconv.ParseInt(s, 10, 64)
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "k"):
		mult = 1024
		s = strings.TrimSuffix(s, "k")
	case strings.HasSuffix(s, "m"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	case strings.HasSuffix(s, "g"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	default:
		return 0, fmt.Errorf("invalid memory format: %s", s)
	}
	if !isDigits(s) {
		return 0, fmt.Errorf("invalid memory number: %s", s)
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return v * mult, nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func (m *Manager) Cleanup() error {
	// best-effort: chỉ remove được khi không còn process trong group
	_ = os.Remove(m.MemPath)
	_ = os.Remove(m.CpuPath)
	_ = os.Remove(m.PidsPath)

	// xoá thư mục cha .../containergo nếu rỗng (best-effort)
	_ = os.Remove(filepath.Dir(m.MemPath))
	_ = os.Remove(filepath.Dir(m.CpuPath))
	_ = os.Remove(filepath.Dir(m.PidsPath))
	return nil
}
