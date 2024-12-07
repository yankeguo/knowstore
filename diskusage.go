package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func calculateDiskUsage(out *int64, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		fullname := filepath.Join(dir, entry.Name())

		stat, err := os.Lstat(fullname)

		if err != nil {
			continue
		}

		if stat.Mode().IsRegular() {
			*out += stat.Size()
		} else if stat.Mode().IsDir() {
			calculateDiskUsage(out, fullname)
		}
	}
	return
}

func prettyDiskUsage(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2fT", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.2fG", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2fM", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2fK", float64(size)/KB)
	default:
		return fmt.Sprintf("%d", size)
	}
}
