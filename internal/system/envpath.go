package system

import (
	"os"
	"strings"
)

type PathEntry struct {
	Path        string `json:"path"`
	IsValid     bool   `json:"is_valid"`
	IsDuplicate bool   `json:"is_duplicate"`
	StatusText  string `json:"status_text"`
}

type EnvVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type EnvPathAnalysis struct {
	TotalEntries    int           `json:"total_entries"`
	ValidEntries    int           `json:"valid_entries"`
	InvalidEntries  int           `json:"invalid_entries"`
	DuplicatesCount int           `json:"duplicates_count"`
	PathEntries     []PathEntry   `json:"path_entries"`
	KeyVariables    []EnvVariable `json:"key_variables"`
}

func GetEnvPathAnalysis() (*EnvPathAnalysis, error) {
	pathVar := os.Getenv("PATH")
	rawPaths := strings.Split(pathVar, string(os.PathListSeparator))

	seenMap := make(map[string]bool)
	entries := []PathEntry{}

	validCount := 0
	invalidCount := 0
	duplicateCount := 0

	for _, p := range rawPaths {
		pClean := strings.TrimSpace(p)
		if pClean == "" {
			continue
		}

		isDuplicate := false
		if seenMap[strings.ToLower(pClean)] {
			isDuplicate = true
			duplicateCount++
		} else {
			seenMap[strings.ToLower(pClean)] = true
		}

		isValid := false
		statusText := "Valid Directory"
		if info, err := os.Stat(pClean); err == nil && info.IsDir() {
			isValid = true
			validCount++
		} else {
			statusText = "Invalid / Missing Directory"
			invalidCount++
		}

		if isDuplicate {
			statusText = "Duplicate Entry"
		}

		entries = append(entries, PathEntry{
			Path:        pClean,
			IsValid:     isValid,
			IsDuplicate: isDuplicate,
			StatusText:  statusText,
		})
	}

	// Key Developer Environment Variables
	keyVarsList := []string{
		"JAVA_HOME", "PYTHONPATH", "GOROOT", "GOPATH", "CUDA_PATH", "NODE_ENV", "TEMP", "TMP",
	}

	keyVars := []EnvVariable{}
	for _, kv := range keyVarsList {
		val := os.Getenv(kv)
		if val != "" {
			// Truncate long variable values
			if len(val) > 60 {
				val = val[:57] + "..."
			}
			keyVars = append(keyVars, EnvVariable{
				Key:   kv,
				Value: val,
			})
		}
	}

	return &EnvPathAnalysis{
		TotalEntries:    len(entries),
		ValidEntries:    validCount,
		InvalidEntries:  invalidCount,
		DuplicatesCount: duplicateCount,
		PathEntries:     entries,
		KeyVariables:    keyVars,
	}, nil
}
