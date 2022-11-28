# HLS encoders control protocol

HLS encodes will connect to the coordinator server vie WebSockets, at the URL:

```
ws(s)://{COORDINATOR_HOST}:{COORDINATOR_PORT}/ws/control/hls
```

In order to authenticate, the HLS encoder will provide the `x-control-auth-token` header. This header will contain a JWT (JSON Web Token) signed with the shared secret provided in the environment variable `CONTROL_SECRET` (shared between the coordinator and the encoders), using the hash signature algorithm `HMAC_256`, with the subject set to `hls-control`.

## Message format

The messages are UTF-8 encoded strings, with parts split by line breaks (\n):
 
  - The first line is the message type
  - After it, the message can have an arbitrary number of arguments. Each argument has a name, followed by a colon and it's value.
  - Optionally, after the arguments, it can be an empty line, followed by the body of the message. In this case, the body will be JSON encoded messages referring to the SDP and candidate exchange.

```
MESSAGE-TYPE
Request-ID: request-id
Auth: auth-token
Argument: value

{body}
...
```

## Message types

Here is the full list of message types, including their purpose and full structure explained.

### Heartbeat

The heartbeat messages are sent each 30 seconds by both, the coordinator and the hls encoder.

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

### Register

When the HLS encoder connects to the coordinator, it must send a `REGISTER` message, giving the coordinator the details to be able to make use of the encoder.

The required arguments are:

 - `Capacity` - Number of video streams the HLS encoder is able to handle in parallel. If it's set to 0, it means there is no limit to enforce.

```
REGISTER

Capacity: 10
```

### Encode-Start

When the coordinator assigns the task of encoding a video stream to the HLS encoder, it will send a `ENCODE-START` message.

The required arguments are:

 - `Stream-Channel` - Unique identifier of the streaming channel
 - `Stream-ID` - Unique identifier of the video stream session
 - `Stream-Source-Type` - Type of source to encode. Can be `RTMP` or `WS`.
 - `Stream-Source-URI` - Source URL to fetch the video stream
 - `Resolutions` - List of playback resolutions. Format: `{WIDTH}x{HEIGHT}-{FPS}`. Split by commas. The encoder will check the source resolution and will encode to at least one resolution (the closest one) and every resolution below this one.
 - `Record` - You can set it to `True` or `False`. Enabling it means the encoder will keep all the HLS fragments, and a separate VOD playlist.
 - `Previews` - Format: `{WIDTH}x{HEIGHT}, {DELAY_SECONDS}` If enabled, the encoder will save a snapshot image of the stream each `DELAY_SECONDS` seconds. Set `Previews: False` to disable it.

```
ENCODE-START

Stream-Channel: example-channel
Stream-ID: example-stream-identifier
Stream-Source-Type: RTMP
Stream-Source-URI: rtmp://127.0.0.1/channel/key
Resolutions: 1280x720-30, 858x480-30
Record: True
Previews: 256x144, 3
```

### Encode-Stop

When the coordinator decides to end a stream, it will send a `ENCODE-STOP` message.

The required arguments are:

 - `Stream-Channel` - Unique identifier of the streaming channel
 - `Stream-ID` - Unique identifier of the video stream session
 - `Grace-Period` - Number of seconds to wait for the encoding process to finish. Set it to 0 to end the stream immediately.

```
ENCODE-STOP

Stream-Channel: example-channel
Stream-ID: example-stream-identifier
Grace-Period: 30
```

### Stream-Available

When the encoder makes the first video segment available for any video stream, it will send a `STREAM-AVAILABLE` message.

It will send a message for each stream type and resolution, since they can be available at different times.

The required arguments are:

 - `Stream-Channel` - Unique identifier of the streaming channel
 - `Stream-ID` - Unique identifier of the video stream session
 - `Stream-Type` - Type of stream. Can be `HLS-LIVE`, `HLS-VOD` or `IMG-PREVIEW`.
 - `Resolution` - Resolution with format `{WIDTH}x{HEIGHT}-{FPS}`
 - `Index-file` - Full path to the index file in the shared file system. It can be a `m3u8` playlist or a `json` file for the images.

```
STREAM-AVAILABLE

Stream-Channel: example-channel
Stream-ID: example-stream-identifier
Stream-Type: HLS-LIVE
Resolution: 1280x720-30
Index-file: {Stream-Channel}/{Stream-ID}/hls/1280x720-30/live.m3u8
```

### Stream-Closed

When the encoding process finished, either normally or due to an error, the encoder will send a `STREAM-CLOSED` message.

The required arguments are:

 - `Stream-Channel` - Unique identifier of the streaming channel
 - `Stream-ID` - Unique identifier of the video stream session

Optional arguments are:

 - `Error-Code` - Error code
 - `Error-Message` - Error message or description

```
STREAM-CLOSED

Stream-Channel: example-channel
Stream-ID: example-stream-identifier
```