# Coordinator HTTP API specification

In order to communicate between your web application and the streaming system, they will use HTTP requests.

There are 3 kind of requests:

 - Streaming key verification requests: For each publishing request, the coordinator will send a request to the application in order for it to verify the key.
 - Event callbacks: The coordinator will send HTTP requests for streaming events (when a streaming is available, or it is closed).
 - Commands: If the application needs to perform some actions, like closing an stream, it must send a request to the coordinator.

## Streaming key verification requests

In order to stablish a mechanism for verifying streaming keys, your application must implement an API to do so.

You must set the `KEY_VERIFICATION_URL` environment variable to the URL of the API implemented by your application. If not set, the coordinator will accept any key.

The request is a **POST** HTTP request, with an **empty body**, and the following **headers**:

 - `x-streaming-channel`: Unique identifier of the streaming channel.
 - `x-streaming-key`: Streaming key used by the publisher.
 - `x-user-ip`: IP address of the user.
 - `Authorization`: Authorization header, depending on your auth method.

If you require authorization for your API, you can use any of the following options (Set for the `KEY_VERIFICATION_AUTH` environment variable):

 - `Basic` - Basic HTTP authorization. Set `KEY_VERIFICATION_AUTH_USER` and `KEY_VERIFICATION_PASSWORD` environment variables.
 - `Bearer` - Bearer token authentication. Set `KEY_VERIFICATION_AUTH_TOKEN` environment variable.
 - `Custom` - Custom authentication header. Set `KEY_VERIFICATION_AUTH_CUSTOM` environment variable.

The API must end the request with status code **200** if the key is valid. Any other status code will result in the publishing session to be closed.

The API may return with the following headers, in order to customize the stream capabilities:

 - `x-record` - Set to `true` or `false` to enable or disable stream recording.
 - `x-previews` - Format: `{WIDTH}x{HEIGHT}, {DELAY_SECONDS}` If enabled, the encoder will save a snapshot image of the stream each `DELAY_SECONDS` seconds. Set `Previews: False` to disable it.
 - `x-resolutions` - List of playback resolutions. Format: `{WIDTH}x{HEIGHT}-{FPS}` or `ORIGINAL`. Split by commas. The encoder will check the source resolution and will encode to at least one resolution (the closest one) and every resolution below this one.

## Event callbacks

In order to process streaming events, your application must implement an API to do so.

You must set the `EVENT_CALLBACK_URL` environment variable to the URL of the API implemented by your application. If not set, the coordinator won't send any event callbacks.

The request is a **POST** HTTP request, with an **empty body**, and the following **headers**:

 - `x-streaming-channel`: Unique identifier of the streaming channel.
 - `x-streaming-id`: Unique identifier of the streaming session.
 - `x-event-type`: Event type. Can be `stream-available` if the streaming session is available for playback, or `stream-closed` if the streaming session has ended.
 - `x-stream-type` - For the `stream-available` event, multiple events with the same streaming ID will be sent for each type and resolution. Type can be `HLS-LIVE`, `HLS-VOD` or `IMG-PREVIEW`.
 - `x-resolution` - For the `stream-available` event, multiple events with the same streaming ID will be sent for each type and resolution. Resolution is formatted as `{WIDTH}x{HEIGHT}-{FPS}`
 - `x-index-file` - Only for `stream-available` event. Full path to the index file in the shared file system. It can be a `m3u8` playlist or a `json` file for the images.
 - `Authorization`: Authorization header, depending on your auth method.

If you require authorization for your API, you can use any of the following options (Set for the `EVENT_CALLBACK_AUTH` environment variable):

 - `Basic` - Basic HTTP authorization. Set `EVENT_CALLBACK_AUTH_USER` and `EVENT_CALLBACK_PASSWORD` environment variables.
 - `Bearer` - Bearer token authentication. Set `EVENT_CALLBACK_AUTH_TOKEN` environment variable.
 - `Custom` - Custom authentication header. Set `EVENT_CALLBACK_AUTH_CUSTOM` environment variable.


The API must end the request with status code **200**. Otherwise the event will be re-sent until it is successfully processed by the application.

## Commands

The coordinator implements an API for the application to send commands to.

For the API to require authorization, set the method in the `COMMANDS_API_AUTH`:
 - `Basic` - Basic HTTP authorization. Set `COMMANDS_API_AUTH_USER` and `COMMANDS_API_AUTH_PASSWORD` environment variables.
 - `Bearer` - Bearer token authentication. Set `COMMANDS_API_AUTH_TOKEN` environment variable.

### Capacity

Use this command to get information about the current load and capacity of the encoders.

Send a **GET** request to `http(s)://{COORDINATOR_HOST}:{COORDINATOR_PORT}/commands/capacity`

The API will end with the **200** status code if succeeded. It will fail with the status code **401** if the authorization is not valid.

The body of the request will be a **JSON** with the following properties:

 - `load` - Current load (number of active streams)
 - `capacity` - Current capacity (-1 means infinite)
 - `encoderCount` - Current number of HLS encoders

Example:

```json
{
    "load": 1,
    "capacity": 8,
    "encoderCount": 2
}
```

### Close stream

In order to force-close streaming session, the application must send **POST** requests to `http(s)://{COORDINATOR_HOST}:{COORDINATOR_PORT}/commands/close`, with an **empty body** and the following headers:

 - `x-streaming-channel`: Unique identifier of the streaming channel.
 - `x-streaming-id`: Unique identifier of the streaming session to be closed. Use the `*` wildcard to close all streaming sessions for a channel.

The API will end with the **200** status code if succeeded. It will fail with the status code **401** if the authorization is not valid.

### Report

You can use the report command to fetch more detailed information about the status of the streaming cluster.

Send a **GET** request to `http(s)://{COORDINATOR_HOST}:{COORDINATOR_PORT}/commands/report`

The API will end with the **200** status code if succeeded. It will fail with the status code **401** if the authorization is not valid.

The body of the request will be a **JSON** with the following properties:

 - `activeStreams` - List of active streams. Each item has the following properties:
   - `channel` - Channel ID
   - `streamId` - Stream ID
   - `streamServer` - ID of the streaming server where the stream is being published.
   - `encoder` - ID of the assigned encoder server.
 - `streamingServers` - List of streaming servers. Each item has the following properties:
   - `id` - Server identifier
   - `ip` - Server IP address
   - `port` - Server port
   - `ssl` - True if the server uses SSL
   - `serverType` - Can be either `RTMP` or `WS`
 - `encoders` - List of encoding servers. Each item has the following properties:
   - `id` - Encoder identifier
   - `capacity` - Encoder capacity (-1 means infinite). Number of streams the encoder can handle in parallel
   - `load` - Number of streams currently being handled by the encoder
