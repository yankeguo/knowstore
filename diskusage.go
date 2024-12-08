package main

import (
	"fmt"
	"github.com/moby/sys/mountinfo"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const PrefixUpperDir = "upperdir="

type DiskUsage struct {
	mounts []*mountinfo.Info
}

func NewDiskUsage() *DiskUsage {
	du := &DiskUsage{}
	var err error
	if du.mounts, err = mountinfo.GetMounts(func(info *mountinfo.Info) (skip, stop bool) {
		skip = !strings.HasPrefix(info.FSType, "overlayfs")
		return
	}); err != nil {
		log.Println("Failed to get mountinfo:", err)
	}
	return du
}

func (du *DiskUsage) Calculate(rootfs string) (size int64) {
	for _, mount := range du.mounts {
		if filepath.Clean(mount.Mountpoint) == filepath.Clean(rootfs) {
			for _, item := range strings.Split(mount.Options, ",") {
				item = strings.TrimSpace(item)
				if strings.HasPrefix(item, PrefixUpperDir) {
					upperDir := strings.TrimSpace(strings.TrimPrefix(item, PrefixUpperDir))
					log.Println("Found overlayfs upperdir:", upperDir)
					calculateDiskUsage(&size, upperDir, false)
					return
				}
			}
		}
	}

	calculateDiskUsage(&size, rootfs, true)
	return
}

func calculateDiskUsage(out *int64, dir string, detectKernFS bool) {
	if detectKernFS {
		if isKernelFilesystem(dir) {
			return
		}
	}

	entries, _ := os.ReadDir(dir)

	for _, entry := range entries {
		filename := filepath.Join(dir, entry.Name())

		stat, err := os.Lstat(filename)
		if err != nil {
			continue
		}

		if stat.Mode().IsRegular() {
			size := stat.Size()
			if size > 1024*1024*1024*1024 {
				log.Println("File size is too large:", filename)
			}
			*out += size
		} else if stat.Mode().IsDir() {
			calculateDiskUsage(out, filename, detectKernFS)
		}
	}
	return
}

func formatDiskUsage(size int64) string {
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
