package functions

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const CurrentTag = "v1.0.0"

type Tag struct {
	Name       string `json:"name"`
	ZipballURL string `json:"zipball_url"`
	TarballURL string `json:"tarball_url"`
	Commit     Commit `json:"commit"`
	NodeID     string `json:"node_id"`
}

type Commit struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

type UpdateStatus struct {
	IsUpdateAvailable bool   `json:"is_update_available"`
	NewTag            string `json:"new_tag"`
}

func CheckUpdate() (*UpdateStatus, error) {
	updateStatus := &UpdateStatus{
		IsUpdateAvailable: false,
		NewTag:            "",
	}

	// The Microsoft Store updates its own installs; sending those users to a download page
	// would be wrong (and is against Store policy).
	if IsStorePackage() {
		return updateStatus, nil
	}

	url := "https://api.github.com/repos/ananduremanan/sysinfopro/tags"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching tags: %v", err)
	}
	defer resp.Body.Close()

	var tags []Tag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("error decoding tags: %v", err)
	}

	if len(tags) == 0 {
		return updateStatus, nil
	}

	// Check if CurrentTag exists in the list
	found := false
	for _, tag := range tags {
		if tag.Name == CurrentTag {
			found = true
			break
		}
	}

	if !found {
		updateStatus.IsUpdateAvailable = true
		updateStatus.NewTag = "not_found"
	}

	return updateStatus, nil
}
