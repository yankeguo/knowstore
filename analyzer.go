package main

import (
	"fmt"
	"github.com/moby/sys/mountinfo"
	"github.com/yankeguo/rg"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	FSTypeOverlay        = "overlay"
	OptionPrefixUpperDir = "upperdir="
)

type AnalyzerOptions struct {
	Rootfs    string
	RootDir   string
	StateDir  string
	Namespace string
}

type Analyzer struct {
	rootfs    string
	rootDir   string
	stateDir  string
	namespace string
	mounts    []*mountinfo.Info
}

func NewAnalyzer(opts AnalyzerOptions) *Analyzer {
	an := &Analyzer{
		rootfs:    opts.Rootfs,
		rootDir:   opts.RootDir,
		stateDir:  opts.StateDir,
		namespace: optContainerdNamespace,
	}
	var err error
	if an.mounts, err = mountinfo.GetMounts(nil); err != nil {
		log.Println("Failed to get mountinfo:", err)
	}
	return an
}

func (du *Analyzer) ListTaskIDs() (ids []string, err error) {
	defer rg.Guard(&err)

	for _, entry := range rg.Must(os.ReadDir(filepath.Join(
		du.rootfs, strings.TrimPrefix(du.stateDir, "/"), "io.containerd.runtime.v2.task", du.namespace,
	))) {
		if !entry.IsDir() {
			continue
		}
		ids = append(ids, entry.Name())
	}

	return
}

func (du *Analyzer) GetDiskUsage(taskID string) (size int64) {
	taskRootfs := filepath.Join(
		du.rootfs, strings.TrimPrefix(du.stateDir, "/"), "io.containerd.runtime.v2.task", du.namespace, taskID, "rootfs",
	)
	for _, mount := range du.mounts {
		if mount.FSType == FSTypeOverlay {
			if filepath.Clean(mount.Mountpoint) == filepath.Clean(taskRootfs) {
				for _, opt := range strings.Split(mount.VFSOptions, ",") {
					if opt = strings.TrimSpace(opt); strings.HasPrefix(opt, OptionPrefixUpperDir) {
						upperDir := strings.TrimSpace(strings.TrimPrefix(opt, OptionPrefixUpperDir))
						upperDir = filepath.Join(du.rootfs, strings.TrimPrefix(upperDir, "/"))
						log.Println("Using OverlayFS upperdir:", upperDir)
						calculateDiskUsage(&size, upperDir, false)
						return
					}
				}
			}
		}
	}

	calculateDiskUsage(&size, taskRootfs, true)
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
