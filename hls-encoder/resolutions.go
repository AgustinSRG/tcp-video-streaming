// Resolutions management

package main

import (
	"fmt"
	"strconv"
	"strings"
)

// List of resolutions
type ResolutionList struct {
	hasOriginal bool         // True if original resolution is included
	resolutions []Resolution // Specific resolutions
}

// Stores a resolution
type Resolution struct {
	width  int // Width (px)
	height int // Height (px)
	fps    int // Frames per second (Can be -1 meaning original fps)
}

// Encodes resolution to string
func (r *Resolution) Encode() string {
	if r.fps >= 0 {
		return fmt.Sprint(r.width) + "x" + fmt.Sprint(r.height) + "-" + fmt.Sprint(r.fps)
	} else {
		return fmt.Sprint(r.width) + "x" + fmt.Sprint(r.height)
	}
}

// Encodes a resolution list to string
func (list *ResolutionList) Encode() string {
	str := ""

	if list.hasOriginal {
		str += "ORIGINAL"
	}

	for i := 0; i < len(list.resolutions); i++ {
		if str == "" {
			str += list.resolutions[i].Encode()
		} else {
			str += "," + list.resolutions[i].Encode()
		}
	}

	return str
}

// Decodes a resolution list
// str - Received string
// Returns the resolution list
func DecodeResolutionsList(str string) ResolutionList {
	if str == "" {
		return ResolutionList{
			hasOriginal: true,
			resolutions: make([]Resolution, 0),
		}
	}

	resList := ResolutionList{
		hasOriginal: false,
		resolutions: make([]Resolution, 0),
	}

	list := strings.Split(str, ",")

	for i := 0; i < len(list); i++ {
		fpsParts := strings.Split(strings.Trim(list[i], " "), "-")

		if len(fpsParts) == 2 {
			resParts := strings.Split(fpsParts[0], "x")

			if len(resParts) == 2 {
				width, err := strconv.ParseInt(resParts[0], 10, 32)

				if err != nil {
					continue
				}

				height, err := strconv.ParseInt(resParts[1], 10, 32)

				if err != nil {
					continue
				}

				fps, err := strconv.ParseInt(fpsParts[1], 10, 32)

				if err != nil {
					continue
				}

				resList.resolutions = append(resList.resolutions, Resolution{
					width:  int(width),
					height: int(height),
					fps:    int(fps),
				})
			}
		} else if len(fpsParts) == 1 {
			resParts := strings.Split(fpsParts[0], "x")

			if len(resParts) == 2 {
				width, err := strconv.ParseInt(resParts[0], 10, 32)

				if err != nil {
					continue
				}

				height, err := strconv.ParseInt(resParts[1], 10, 32)

				if err != nil {
					continue
				}

				resList.resolutions = append(resList.resolutions, Resolution{
					width:  int(width),
					height: int(height),
					fps:    -1,
				})
			} else if len(resParts) == 1 && strings.ToUpper(resParts[0]) == "ORIGINAL" {
				resList.hasOriginal = true
			}
		}
	}

	return resList
}

// Stream previews configuration
type PreviewsConfiguration struct {
	enabled      bool // True if enabled
	width        int  // Width (px)
	height       int  // Height (px)
	delaySeconds int  // Delay (seconds)
}

// Encodes previews configuration to string
func (pc *PreviewsConfiguration) Encode() string {
	if pc.enabled {
		return fmt.Sprint(pc.width) + "x" + fmt.Sprint(pc.height) + "," + fmt.Sprint(pc.delaySeconds)
	} else {
		return "False"
	}
}

// Decodes previews configuration from string
// str - String to parse
// Returns the configuration params
func DecodePreviewsConfiguration(str string) PreviewsConfiguration {
	delayParts := strings.Split(str, ",")

	if len(delayParts) == 2 {
		delaySeconds, err := strconv.ParseInt(strings.Trim(delayParts[1], " "), 10, 32)

		if err != nil || delaySeconds < 1 {
			return PreviewsConfiguration{
				enabled: false,
			}
		}

		resParts := strings.Split(strings.Trim(delayParts[0], " "), "x")

		if len(resParts) != 2 {
			width, err := strconv.ParseInt(resParts[0], 10, 32)

			if err != nil {
				return PreviewsConfiguration{
					enabled: false,
				}
			}

			height, err := strconv.ParseInt(resParts[1], 10, 32)

			if err != nil {
				return PreviewsConfiguration{
					enabled: false,
				}
			}

			return PreviewsConfiguration{
				enabled:      true,
				width:        int(width),
				height:       int(height),
				delaySeconds: int(delaySeconds),
			}
		} else {
			return PreviewsConfiguration{
				enabled: false,
			}
		}
	} else {
		return PreviewsConfiguration{
			enabled: false,
		}
	}
}
