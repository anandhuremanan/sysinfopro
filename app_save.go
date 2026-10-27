package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
)

// downloadsDir is where cards and reports are saved: the user's real Downloads
// folder (which may have been moved to another drive or into OneDrive), else
// the profile folder
func downloadsDir() string {
	if dir := knownDownloads(); dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	if dir := filepath.Join(home, "Downloads"); dirExists(dir) {
		return dir
	}
	return home
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// isPNG checks the file signature, so only a PNG image is ever saved as one
func isPNG(data []byte) bool {
	return bytes.HasPrefix(data, pngSignature)
}

// reportCSP stops the saved report from loading or running anything: it is
// plain text and styles, opened later in a browser from the Downloads folder
const reportCSP = `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src data:">`

var headTag = regexp.MustCompile(`(?i)<head[^>]*>`)

// withCSP puts the content security policy at the start of the report's head
func withCSP(html string) string {
	if loc := headTag.FindStringIndex(html); loc != nil {
		return html[:loc[1]] + "\n" + reportCSP + html[loc[1]:]
	}
	return reportCSP + "\n" + html
}
