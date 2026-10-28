//go:build bindings

package main

// Wails builds the app with the "bindings" tag and runs it once, during
// `wails build`, `wails dev` and `wails generate module`, only to list the Go
// functions the screens can call. That run must not touch the user's files.
const generatingBindings = true
