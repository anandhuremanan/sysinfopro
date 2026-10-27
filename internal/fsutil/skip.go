package fsutil

import "strings"

// commonSkip lists folders that whole-drive searches never walk into: Windows'
// own folders, installed programs and app data, and the dependency and build
// caches tools rebuild on their own. Removing a "duplicate" from them breaks a
// program or a project, and walking them costs most of a search's time.
var commonSkip = map[string]bool{
	"windows": true, "program files": true, "program files (x86)": true, "programdata": true,
	"$recycle.bin": true, "system volume information": true, "recovery": true, "appdata": true,
	"node_modules": true, ".venv": true, "venv": true, "__pycache__": true, "site-packages": true,
	".gradle": true, ".m2": true, ".nuget": true, ".cargo": true, ".rustup": true, "target": true,
	"vendor": true, "obj": true, ".pnpm-store": true, ".npm": true, ".cache": true,
}

// SkipFolder reports whether a search should not walk into a folder called name:
// one of the common folders above, or one of the caller's own extras (given in
// lower case)
func SkipFolder(name string, extra map[string]bool) bool {
	n := strings.ToLower(name)
	return commonSkip[n] || extra[n]
}
