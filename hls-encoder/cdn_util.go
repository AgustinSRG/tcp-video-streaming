// HLS websocket CDN utils

package main

import (
	"net/url"
	"strings"
)

// Websocket protocol message
type CdnWebsocketProtocolMessage struct {
	// Message type
	MessageType string

	// Message parameters
	Parameters map[string]string
}

// Gets the parameter value
func (msg *CdnWebsocketProtocolMessage) GetParameter(param string) string {
	if msg.Parameters == nil {
		return ""
	}

	return msg.Parameters[param]
}

// Serializes message to string (to be sent)
func (msg *CdnWebsocketProtocolMessage) Serialize() string {
	if msg.Parameters == nil || len(msg.Parameters) == 0 {
		return msg.MessageType
	}

	paramStr := ""

	for k, v := range msg.Parameters {
		if len(paramStr) > 0 {
			paramStr += "&"
		}

		paramStr += url.QueryEscape(k) + "=" + url.QueryEscape(v)
	}

	return msg.MessageType + ":" + paramStr
}

// Parses websocket protocol message from string
func ParseCdnWebsocketProtocolMessage(str string) *CdnWebsocketProtocolMessage {
	colonIndex := strings.IndexRune(str, ':')

	if colonIndex < 0 || colonIndex >= len(str)-1 {
		return &CdnWebsocketProtocolMessage{
			MessageType: strings.ToUpper(str),
		}
	}

	msgType := strings.ToUpper(str[0:colonIndex])
	msgParams := str[colonIndex+1:]

	q, err := url.ParseQuery(msgParams)

	if err != nil {
		return &CdnWebsocketProtocolMessage{
			MessageType: msgType,
		}
	}

	params := make(map[string]string)

	for k, v := range q {
		params[k] = strings.Join(v, "")
	}

	return &CdnWebsocketProtocolMessage{
		MessageType: msgType,
		Parameters:  params,
	}
}
