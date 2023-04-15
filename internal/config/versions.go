package config

import (
	"os"
	"strings"
)

const versionsFile = "versions.txt"

func GetVersions() ([]string, error) {
	versionsFile, err := os.ReadFile(versionsFile)
	if err != nil {
		return nil, err
	}

	versions := strings.Split(strings.TrimSpace(string(versionsFile)), "\n")

	return versions, nil
}
