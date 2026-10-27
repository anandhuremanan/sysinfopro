package main

import (
	"strings"
	"testing"
)

func TestIsPNG(t *testing.T) {
	if !isPNG([]byte("\x89PNG\r\n\x1a\nrest")) {
		t.Error("a PNG should be accepted")
	}
	for _, bad := range []string{"", "GIF89a", "<html>", "\x89PNG"} {
		if isPNG([]byte(bad)) {
			t.Errorf("%q is not a PNG", bad)
		}
	}
}

func TestWithCSP(t *testing.T) {
	got := withCSP("<!doctype html><html><HEAD lang=en><title>x</title></head><body></body></html>")
	head := strings.Index(got, "<HEAD lang=en>")
	csp := strings.Index(got, "Content-Security-Policy")
	title := strings.Index(got, "<title>")
	if head < 0 || csp < head || csp > title {
		t.Errorf("policy should come first in the head: %s", got)
	}
	if !strings.Contains(withCSP("<p>no head</p>"), "Content-Security-Policy") {
		t.Error("a page without a head still gets the policy")
	}
}

func TestDownloadsDirExists(t *testing.T) {
	if !dirExists(downloadsDir()) {
		t.Error("downloads folder should exist")
	}
}
