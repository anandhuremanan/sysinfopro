//go:build !windows

package main

func windowsTickMs() uint64 { return 0 }
