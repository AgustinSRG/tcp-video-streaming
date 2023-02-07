// Resolutions management

package main

import (
	"fmt"
	"math"
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
// delimiter - The delimiter (size - delay)
func (pc *PreviewsConfiguration) Encode(delimiter string) string {
	if pc.enabled {
		return fmt.Sprint(pc.width) + "x" + fmt.Sprint(pc.height) + delimiter + fmt.Sprint(pc.delaySeconds)
	} else {
		return "False"
	}
}

// Decodes previews configuration from string
// str - String to parse
// delimiter - The delimiter (size - delay)
// Returns the configuration params
func DecodePreviewsConfiguration(str string, delimiter string) PreviewsConfiguration {
	delayParts := strings.Split(str, delimiter)

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

// Gets resolution list for encoding
// originalResolution - Original resolution
// list - Configured resolutions list
// Returns a list of resolutions
func GetActualResolutionList(originalResolution Resolution, list ResolutionList) []Resolution {
	result := make([]Resolution, 0)
	var smallerResolution *Resolution = nil
	resultSet := make(map[string]bool)

	if list.hasOriginal {
		resultSet[originalResolution.Encode()] = true
	}

	originalWidth := originalResolution.width

	if originalWidth <= 0 {
		originalWidth = 1
	}

	originalHeight := originalResolution.height

	if originalHeight <= 0 {
		originalHeight = 1
	}

	originalFPS := originalResolution.fps

	if originalFPS <= 0 {
		originalFPS = 30
	}

	for i := 0; i < len(list.resolutions); i++ {
		fitWidth := list.resolutions[i].width

		if fitWidth <= 0 {
			fitWidth = 1
		}

		if fitWidth%2 != 0 {
			fitWidth++
		}

		fitHeight := list.resolutions[i].height

		if fitHeight <= 0 {
			fitHeight = 1
		}

		if fitHeight%2 != 0 {
			fitHeight++
		}

		fitFPS := list.resolutions[i].fps

		if fitFPS <= 0 {
			fitFPS = originalFPS
		}

		finalResolution := Resolution{
			width:  fitWidth,
			height: fitHeight,
			fps:    fitFPS,
		}

		if originalFPS < fitFPS {
			finalResolution.fps = originalFPS
		}

		proportionalHeight := int(math.Ceil((float64(originalHeight)*float64(fitWidth)/float64(originalWidth))/2) * 2)
		proportionalWidth := int(math.Ceil((float64(originalWidth)*float64(fitHeight)/float64(originalHeight))/2) * 2)

		if originalWidth > originalHeight {
			if proportionalHeight > fitHeight {
				finalResolution.width = proportionalWidth
			} else {
				finalResolution.height = proportionalHeight
			}
		} else {
			if proportionalWidth > fitWidth {
				finalResolution.height = proportionalHeight
			} else {
				finalResolution.width = proportionalWidth
			}
		}

		resolutionId := finalResolution.Encode()

		if resultSet[resolutionId] {
			continue
		}

		resultSet[resolutionId] = true

		if smallerResolution == nil || (finalResolution.width <= smallerResolution.width && finalResolution.height <= smallerResolution.height && finalResolution.fps <= smallerResolution.fps) {
			smallerResolution = &finalResolution
		}

		if finalResolution.height <= originalHeight && finalResolution.width <= originalWidth {
			// Smaller, append
			result = append(result, finalResolution)
		}
	}

	if len(result) > 0 {
		return result
	} else if !list.hasOriginal && smallerResolution != nil {
		return []Resolution{{
			width:  smallerResolution.width,
			height: smallerResolution.height,
			fps:    smallerResolution.fps,
		}}
	} else {
		return result
	}
}
