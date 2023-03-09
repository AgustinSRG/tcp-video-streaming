// Messages

package messages

import "strings"

// Websocket message
type WebsocketMessage struct {
	Method string            // Message type / method
	Params map[string]string // Message arguments
	Body   string            // Message body (extra data)
}

// Parses websocket message from string message received
// raw - Raw string to parse
func ParseWebsocketMessage(raw string) WebsocketMessage {
	lines := strings.Split(raw, "\n")
	msg := WebsocketMessage{
		Method: "",
		Params: make(map[string]string),
		Body:   "",
	}

	if len(lines) > 0 {
		msg.Method = strings.ToUpper(strings.Trim(lines[0], " \n\r\t"))
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
				msg.Body = line
				firstLineBody = false
			} else {
				msg.Body += "\n" + line
			}
		} else {
			// Param
			colonIndex := strings.Index(line, ":")
			if colonIndex > 0 {
				key := strings.ToLower(strings.Trim(line[0:colonIndex], " \n\r\t"))
				val := strings.Trim(line[colonIndex+1:], " \n\r\t")
				msg.Params[key] = val
			}
		}
	}

	return msg
}

// Gets a param from the message
// paramName - Name of the param
// Returns the param value
func (s WebsocketMessage) GetParam(paramName string) string {
	if s.Params == nil {
		return ""
	}
	return s.Params[strings.ToLower(paramName)]
}

// Serializes websocket message in order to send it
func (s WebsocketMessage) Serialize() string {
	var raw string
	raw = strings.ToUpper(s.Method) + "\n"

	if s.Params != nil {
		for key, val := range s.Params {
			raw += key + ":" + val + "\n"
		}
	}

	if s.Body != "" {
		raw += "\n" + s.Body
	}

	return raw
}
