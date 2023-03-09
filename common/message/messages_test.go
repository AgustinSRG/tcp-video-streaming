// Tests

package messages

import "testing"

func TestNoParamsMessage(t *testing.T) {
	message := WebsocketMessage{Method: "TEST"}

	serialized := message.Serialize()

	recovered := ParseWebsocketMessage(serialized)

	if recovered.Method != "TEST" {
		t.Errorf("Invalid method")
	}

	if recovered.GetParam("test-param") != "" {
		t.Errorf("Invalid param")
	}
}

func TestNoBodyMessage(t *testing.T) {
	message := WebsocketMessage{Method: "TEST", Params: map[string]string{"Test-Param": "Test-Value", "Test-Param-2": "Test-Value-2"}}

	serialized := message.Serialize()

	recovered := ParseWebsocketMessage(serialized)

	if recovered.Method != "TEST" {
		t.Errorf("Invalid method")
	}

	if recovered.GetParam("test-param") != "Test-Value" {
		t.Errorf("Invalid parameter (1)")
	}

	if recovered.GetParam("test-param-2") != "Test-Value-2" {
		t.Errorf("Invalid parameter (2)")
	}
}

func TestFullMessage(t *testing.T) {
	message := WebsocketMessage{Method: "TEST", Params: map[string]string{"Test-Param": "Test-Value", "Test-Param-2": "Test-Value-2"}, Body: "Test Body\nTest second line\nThird line"}

	serialized := message.Serialize()

	recovered := ParseWebsocketMessage(serialized)

	if recovered.Method != "TEST" {
		t.Errorf("Invalid method")
	}

	if recovered.GetParam("test-param") != "Test-Value" {
		t.Errorf("Invalid parameter (1)")
	}

	if recovered.GetParam("test-param-2") != "Test-Value-2" {
		t.Errorf("Invalid parameter (2)")
	}

	if recovered.Body != "Test Body\nTest second line\nThird line" {
		t.Errorf("Invalid body")
	}
}
