// Websocket messages

package main

import "strings"

// Websocket message
type WebsocketMessage struct {
	method string
	params map[string]string
	body   string
}

// Parses websocket message from string message received
func parseWebsocketMessage(raw string) WebsocketMessage {
	lines := strings.Split(raw, "\n")
	msg := WebsocketMessage{
		method: "",
		params: make(map[string]string),
		body:   "",
	}

	if len(lines) > 0 {
		msg.method = strings.ToUpper(strings.Trim(lines[0], " \n\r\t"))
	}

	var isBody bool = false
	var firstLineBody bool = true

	for i := 1; i < len(lines); i++ {
		line := lines[i]

		if line == "" {
			// Found empty line
			isBody = true
			continue
		}

		if isBody {
			// Body
			if firstLineBody {
				msg.body = line
				firstLineBody = false
			} else {
				msg.body += "\n" + line
			}
		} else {
			// Param
			colonIndex := strings.Index(line, ":")
			if colonIndex > 0 {
				key := strings.ToLower(strings.Trim(line[0:colonIndex], " \n\r\t"))
				val := strings.Trim(line[colonIndex+1:], " \n\r\t")
				msg.params[key] = val
			}
		}
	}

	return msg
}

// Gets a param from the message
// paramName - Name of the param
// Returns the param value
func (s WebsocketMessage) GetParam(paramName string) string {
	return s.params[strings.ToLower(paramName)]
}

// Serializes websocket message in order to send it
func (s WebsocketMessage) serialize() string {
	var raw string
	raw = strings.ToUpper(s.method) + "\n"

	if s.params != nil {
		for key, val := range s.params {
			raw += key + ":" + val + "\n"
		}
	}

	if s.body != "" {
		raw += "\n" + s.body
	}

	return raw
}
