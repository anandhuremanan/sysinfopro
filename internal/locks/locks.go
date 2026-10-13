// Package locks finds the processes that hold a file or folder open.
package locks

// most files a folder lookup will check; Windows slows down sharply beyond this
const maxFilesChecked = 3000

type LockingProcess struct {
	PID  uint32 `json:"pid"`
	Name string `json:"name"`
	// What kind of program it is, e.g. "Application", "Service", "Explorer"
	Kind string `json:"kind"`
}

type Result struct {
	Path         string `json:"path"`
	IsDir        bool   `json:"is_dir"`
	FilesChecked int    `json:"files_checked"`
	// true when a folder held more files than could be checked
	Truncated bool             `json:"truncated"`
	Processes []LockingProcess `json:"processes"`
}
