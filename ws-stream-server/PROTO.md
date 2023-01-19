# Websocket streaming server protocol

Clients must connect using the websocket protocol to an URL like this:
```
ws://{HOST}:{PORT}/{CHANNEL}/{KEY}/{CONNECTION-KIND}
```

The `CONNECTION-KIND` can be: `publish`, `receive`, `receive-clear-cache` or `probe`.

Note: Both `CHANNEL` and `KEY` are restricted to letters `a-z`, numbers `0-9`, dashes `-` and underscores `_`.

## Heartbeat

In order for the server and client to keep the connection alive, both sides will send heartbeat messages periodically.

A heartbeat message consiste of a **TEXT** websocket message containing the character `h`.

If no heartbeat message is received during a period larger than 1 minute, the connection can be terminated due to inactivity.

## Error

If an error happens, for example the key provided is invalid, an error message is sent, and right after, the connection is terminated.

An error message consists of a **TEXT** websocket message starting with the `ERROR:` prefix. Example:

```
ERROR: Invalid streaming key
```

## Publish

For publishing, the client will connect using a `CONNECTION-KIND` = `publish`.

If the key is allowed and there are no other clients publishing for the same channel, the websocket connection will be accepted.

Once the connection is opened, the client must send the data stream as **BINARY** websocket messages.

When finishing publishing, the connection must be closed by the client.

## Receive

For receiving, the client will connect using a `CONNECTION-KIND` = `receive`. Alternatively, you can use `receive-clear-cache` for clearing the GOP cache after connecting.

After connecting, if there is no publisher yet, it will wait for one.

The data chunks will be received as soon as available as **BINARY** websocket messages.

If there are chunks in the cache, those will be received immediately after connecting.

When the publisher finishes, the connection will be closed by the server.

## Probe

For probing, the client will connect using a `CONNECTION-KIND` = `probe`.

After connecting, if there is no publisher yet, it will wait for one.

As soon as a single chunk is available, it will be received as as **BINARY** websocket message.

After receiving a single chunk, the connection will be closed by the server.

This kid of connection is very useful for checking the data, for example to detect the codecs used for video and audio, or the resolution of the video.
