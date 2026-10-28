//go:build !windows

package main

const restartFlag = "--after-restart"

func waitForRestart() {}
