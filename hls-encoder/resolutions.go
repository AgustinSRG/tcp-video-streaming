// Resolutions management

package main

import (
	"errors"
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
	width   int // Width (px)
	height  int // Height (px)
	fps     int // Frames per second (Can be -1 meaning original fps)
	bitRate int // Bitrate limit in kilobits per second (Can be -1 meaning no limit)
}

// Encodes resolution to string
func (r *Resolution) Encode() string {
	str := fmt.Sprint(r.width) + "x" + fmt.Sprint(r.height)

	if r.fps > 0 {
		str += "-" + fmt.Sprint(r.fps)
	}

	if r.bitRate > 0 {
		str += "~" + fmt.Sprint(r.bitRate)
	}

	return str
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

// Decodes resolution
// str - Encoded resolution
// Returns the decoded resolution
func DecodeResolution(str string) (resolution Resolution, resErr error) {
	bitRateParts := strings.Split(strings.TrimSpace(str), "~")
	bitRate := -1

	if len(bitRateParts) == 2 {
		p, err := strconv.ParseInt(bitRateParts[1], 10, 32)

		if err != nil {
			return Resolution{}, err
		}

		bitRate = int(p)
	}

	str = bitRateParts[0]

	fpsParts := strings.Split(strings.TrimSpace(str), "-")
	fps := -1

	if len(fpsParts) == 2 {
		p, err := strconv.ParseInt(fpsParts[1], 10, 32)

		if err != nil {
			return Resolution{}, err
		}

		fps = int(p)
	}

	str = fpsParts[0]

	resParts := strings.Split(str, "x")

	if len(resParts) != 2 {
		return Resolution{}, errors.New("invalid resolution")
	}

	width, err := strconv.ParseInt(resParts[0], 10, 32)

	if err != nil {
		return Resolution{}, err
	}

	height, err := strconv.ParseInt(resParts[1], 10, 32)

	if err != nil {
		return Resolution{}, err
	}

	return Resolution{
		width:   int(width),
		height:  int(height),
		fps:     fps,
		bitRate: bitRate,
	}, nil
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

		el := list[i]

		if strings.ToUpper(el) == "ORIGINAL" {
			resList.hasOriginal = true
			continue
		}

		r, err := DecodeResolution(el)

		if err != nil {
			continue
		}

		resList.resolutions = append(resList.resolutions, r)
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
		delaySeconds, err := strconv.ParseInt(strings.TrimSpace(delayParts[1]), 10, 32)

		if err != nil || delaySeconds < 1 {
			return PreviewsConfiguration{
				enabled: false,
			}
		}

		resParts := strings.Split(strings.TrimSpace(delayParts[0]), "x")

		if len(resParts) == 2 {
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
			width:   fitWidth,
			height:  fitHeight,
			fps:     fitFPS,
			bitRate: list.resolutions[i].bitRate,
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
			width:   smallerResolution.width,
			height:  smallerResolution.height,
			fps:     smallerResolution.fps,
			bitRate: smallerResolution.bitRate,
		}}
	} else {
		return result
	}
}
