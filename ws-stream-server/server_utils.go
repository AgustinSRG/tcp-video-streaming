// Server utils

package main

import (
	"os"
	"regexp"
	"strconv"
)

var ID_MAX_LENGTH = 128
var idCustomMaxLength = os.Getenv("ID_MAX_LENGTH")

func validateStreamIDString(str string) bool {
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
