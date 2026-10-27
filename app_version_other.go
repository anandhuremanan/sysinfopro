//go:build !windows

package main

func windowsRelease() (name, display, build string, ubr uint64) { return "", "", "", 0 }
