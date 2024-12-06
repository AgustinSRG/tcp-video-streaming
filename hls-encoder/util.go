// Utils

package main

import "strings"

// Get extension from file path
func getFileExtension(path string) string {
	parts := strings.Split(path, "/")

	if len(parts) == 0 {
		return ""
	}

	file := parts[len(parts)-1]

	if !strings.Contains(file, ".") {
		return ""
	}

	fileParts := strings.Split(file, ".")

	if len(fileParts) == 0 {
		return ""
	}

	return fileParts[len(fileParts)-1]
}
