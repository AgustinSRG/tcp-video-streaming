# RTMP servers control protocol

RTMP servers will connect to the coordinator server via WebSockets, at the URL:

```
ws(s)://{COORDINATOR_HOST}:{COORDINATOR_PORT}/ws/control/rtmp
```

In order to authenticate, the RTMP server will provide the `x-control-auth-token` header. This header will contain a JWT (JSON Web Token) signed with the shared secret provided in the environment variable `CONTROL_SECRET` (shared between the coordinator and the RTMP servers), using the hash signature algorithm `HMAC_256`, with the subject set to `rtmp-control`.

## Message format

The messages are UTF-8 encoded strings, with parts split by line breaks (\n):
 
  - The first line is the message type
  - After it, the message can have an arbitrary number of arguments. Each argument has a name, followed by a colon and it's value.
  - Optionally, after the arguments, it can be an empty line, followed by the body of the message.

```
MESSAGE-TYPE
Request-ID: request-id
Auth: auth-token
Argument: value

{body}
```

## Message types

Here is the full list of message types, including their purpose and full structure explained.

### Heartbeat

The heartbeat messages are sent each 30 seconds by both, the coordinator and the rtmp server.

If one of the agents does not receive a heartbeat message during 1 minute, the connection may be closed due to inactivity.

The message does not take any arguments or body.

```
HEARTBEAT
```

### Error

If an error happens, an `ERROR` message will be sent, with the details of the error in the arguments.

The required arguments are:

 - `Error-Code` - Error code
 - `Error-Message` - Error message or description

```
ERROR
Error-Code: EXAMPLE_CODE
Error-Message: Example Error message
```

After this message is received, the connection will be closed.

### Publish-Request

When the RTMP server receives a publish request, it will forward the information to the coordinator with a `PUBLISH-REQUEST` message.

The required arguments are:

 - `Request-ID` - Unique request ID. This will be used to reply, accepting or denying the request.
 - `Stream-Channel` - Unique identifier of the streaming channel
 - `Stream-Key` - Streaming key (used for authentication)
 - `User-IP` - User IP address

```
PUBLISH-REQUEST
Request-ID: 1
Stream-Channel: example-channel
Stream-Key: my-key
User-IP: 127.0.0.1
```

### Publish-Accept

If the coordinator accepts the publish request, it will send a `PUBLISH-ACCEPT` message.

The required arguments are:

 - `Request-ID` - Unique request ID. The same used in the `PUBLISH-REQUEST` message
 - `Stream-Channel` - Unique identifier of the streaming channel
 - `Stream-ID` - Unique identifier of the video stream session


```
PUBLISH-ACCEPT
Request-ID: 1
Stream-Channel: example-channel
Stream-ID: example-stream-identifier
```

### Publish-Deny

If the coordinator rejects the publish request, it will send a `PUBLISH-DENY` message.

The required arguments are:

 - `Request-ID` - Unique request ID. The same used in the `PUBLISH-REQUEST` message
 - `Stream-Channel` - Unique identifier of the streaming channel

```
PUBLISH-DENY
Request-ID: 1
Stream-Channel: example-channel
```

### Publish-End

After being accepted, if a publishing connection ends, the RTMP server will send a `PUBLISH-END` message.

The required arguments are:

 - `Stream-Channel` - Unique identifier of the streaming channel
 - `Stream-ID` - Unique identifier of the video stream session

```
PUBLISH-END
Stream-Channel: example-channel
Stream-ID: example-stream-identifier
```

### Stream-Kill

If the coordinator wants to close an active publishing session, it will send a `STREAM-KILL` message.

The required arguments are:

 - `Stream-Channel` - Unique identifier of the streaming channel
 - `Stream-ID` - Unique identifier of the video stream session. It can be the `*` wildcard, to kill any active video streams for that channel.

```
STREAM-KILL
Stream-Channel: example-channel
Stream-ID: *
```
