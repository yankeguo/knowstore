//go:build !linux

package main

func isKernelFilesystem(filename string) (ok bool) {
	return false
}
