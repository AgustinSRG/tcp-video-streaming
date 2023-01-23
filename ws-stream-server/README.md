# Websocket streaming server

This server implements an alternative protocol to RTMP. It uses websocket connections, allowing clients to publish and receive binary data in real time.

## Compilation

In order to install dependencies, type:

```
go get .
```

To compile the code type:

```
go build
```

The build command will create a binary in the current directory, called `ws-stream-server`, or `ws-stream-server.exe` if you are using Windows.

## Usage

In order to run the server, run the binary without arguments (check the configuration below for customization).

Clients must connect using the websocket protocol to an URL like this:
```
ws://{HOST}:{PORT}/{CHANNEL}/{KEY}/{CONNECTION-KIND}
```

The `CONNECTION-KIND` can be: `publish`, `receive`, `receive-clear-cache` or `probe`. Check the [PROTOCOL](./PROTO.md) for more details.

Note: Both `CHANNEL` and `KEY` are restricted to letters `a-z`, numbers `0-9`, dashes `-` and underscores `_`.

By default, it will accept any connection with any key. Check the configuration section in order to setup the server to be able to work with a coordinator server.

### Configuration

You can configure the server with environment variables.

| Variable Name    | Description                                                                                                                                                                                                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CONTROL_BASE_URL | Websocket URL to connect to the  coordinator server. Example: `wss://10.0.0.0:8080/`                                                                                                                                                                                              |
| CONTROL_SECRET   | Secret shared between the coordinator server and the RTMP server, in order to authenticate.                                                                                                                                                                                       |
| PLAY_WHITELIST   | List of internet addresses allowed to play the data stream. Split by commas. Example: `127.0.0.1,10.0.0.0/8`. You can set IPs, or subnets. It supports both IP version 4 and version 6. This list must include the HLS encoders in order for them to be able to fetch the stream. |

### TLS

If you want to use TLS, you have to set 3 variables in order for it to work:

| Variable Name | Description                            |
| ------------- | -------------------------------------- |
| SSL_PORT      | HTTPS listening port. Default is `443` |
| SSL_CERT      | Path to SSL certificate.               |
| SSL_KEY       | Path to SSL private key.               |

### More options

Here is a list with more options you can configure:

| Variable Name                 | Description                                                                                                                        |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| HTTP_PORT                     | HTTP listening port. Default is `80`                                                                                               |
| BIND_ADDRESS                  | Bind address for RTMP and RTMPS. By default it binds to all network interfaces.                                                    |
| RTMP_CHUNK_SIZE               | RTMP Chunk size in bytes. Default is `128`                                                                                         |
| LOG_REQUESTS                  | Set to `YES` or `NO`. By default is `YES`                                                                                          |
| LOG_DEBUG                     | Set to `YES` or `NO`. By default is `NO`                                                                                           |
| ID_MAX_LENGTH                 | Max length for `CHANNEL` and `KEY`. By default is 128 characters                                                                   |
| MAX_IP_CONCURRENT_CONNECTIONS | Max number of concurrent connections to accept from a single IP. By default is 4.                                                  |
| CONCURRENT_LIMIT_WHITELIST    | List of IP ranges not affected by the max number of concurrent connections limit. Split by commas. Example: `127.0.0.1,10.0.0.0/8` |
| GOP_CACHE_SIZE_MB             | Size limit in megabytes of packet cache. By default is `256`. Set it to `0` to disable cache                                       |
| EXTERNAL_IP                   | In case the server is separated from the rest of the components via NAT or a proxy. Use this to set the external IP address        |
| EXTERNAL_PORT                 | In case the server is separated from the rest of the components via NAT or a proxy. Use this to set the external port number       |
| DISABLE_TEST_CLIENT           | Set to `YES` to disable the default test client (for production)                                                                   |
