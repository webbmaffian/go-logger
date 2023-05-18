package main

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

var ErrNotDir = errors.New("path is not a directory")

const (
	PermPrivDir  fs.FileMode = 0700
	PermPrivFile fs.FileMode = 0600
)

func SetPrivFilePerm(path string, isDir ...bool) (err error) {
	var dir bool
	var perm fs.FileMode = PermPrivFile

	if isDir != nil && isDir[0] {
		dir = true
		perm = PermPrivDir
	}

	info, err := os.Stat(path)

	if err != nil {
		return
	}

	if dir && !info.IsDir() {
		return ErrNotDir
	}

	curUser := os.Getuid()
	curGroup := os.Getgid()

	// Windows will return -1. We can't do anything more, abort without error.
	if curUser == -1 || curGroup == -1 {
		return
	}

	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		if int(stat.Uid) != curUser || int(stat.Gid) != curGroup {
			if err = os.Chown(path, curUser, curGroup); err != nil {
				return
			}
		}
	}

	if info.Mode() != perm {
		err = os.Chmod(path, perm)
	}

	return
}
