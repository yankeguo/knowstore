//go:build linux

package main

import "golang.org/x/sys/unix"

func isKernelFilesystem(filename string) bool {
	var fstat unix.Statfs_t
	if err := unix.Statfs(filename, &fstat); err != nil {
		return false
	}

	switch fstat.Type {
	case unix.PROC_SUPER_MAGIC,
		unix.DEVPTS_SUPER_MAGIC,
		unix.CGROUP_SUPER_MAGIC,
		unix.CGROUP2_SUPER_MAGIC,
		unix.TMPFS_MAGIC,
		unix.TRACEFS_MAGIC,
		unix.SYSFS_MAGIC:
		return true
	}
	return false
}
