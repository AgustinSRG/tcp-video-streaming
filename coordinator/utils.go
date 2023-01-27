// Other utilities and constants

package main

import (
	"os"
	"regexp"
	"strconv"
)

const (
	FILE_PERMISSION   = 0600 // Read/Write
	FOLDER_PERMISSION = 0700 // Read/Write/Run
)

// Validates stream ID
// str - Stream ID
// Returns true only if valid
func validateStreamIDString(str string) bool {
	var ID_MAX_LENGTH = 128
	var idCustomMaxLength = os.Getenv("ID_MAX_LENGTH")

	if idCustomMaxLength != "" {
		var e error
		ID_MAX_LENGTH, e = strconv.Atoi(idCustomMaxLength)
		if e != nil {
			ID_MAX_LENGTH = 128
		}
	}

	if len(str) > ID_MAX_LENGTH {
		return false
	}

	m, e := regexp.MatchString("^[A-Za-z0-9\\_\\-]+$", str)

	if e != nil {
		return false
	}

	return m
}
