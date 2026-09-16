//go:build !windows

package main

import "os"

func replaceCacheFile(source, destination string) error {
	return os.Rename(source, destination)
}
