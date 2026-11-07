//go:build darwin || linux

package main

import (
	"os"
	"syscall"
)

func singleLinkFile(_ string, info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink == 1
}

func clientPathSafe(_ string, info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink == 0
}

func privateConnectionFile(path string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Getuid())
}
