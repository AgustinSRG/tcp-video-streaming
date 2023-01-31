# HLS encoder server

This server acts as a worker, receiving requests from the coordinator server to encode streams to HLS (HTTP Live Streaming).

Note: this component requires for [ffmpeg](https://ffmpeg.org/) to be installed in your system.

## Compilation

In order to install dependencies, type:

```
go get .
```

To compile the code type:

```
go build
```

The build command will create a binary in the current directory, called `hls-encoder`, or `hls-encoder.exe` if you are using Windows.

## Usage

In order to run the server, run the binary without arguments (check the configuration below for customization).

### Configuration

You can configure the server with environment variables.

| Variable Name    | Description                                                                                        |
| ---------------- | -------------------------------------------------------------------------------------------------- |
| SERVER_CAPACITY  | Max number of streams the server can handle in parallel. Set to -1 for unlimited (the default)     |
| CONTROL_BASE_URL | Websocket URL to connect to the  coordinator server. Example: `wss://10.0.0.0:8080/`               |
| CONTROL_SECRET   | Secret shared between the coordinator server and the HLS encoder server, in order to authenticate. |

### Storage

You can configure the storage mode settings the `HLS_STORAGE_MODE` to:

 - `FILE_SYSTEM` - The default value. Set `HLS_FILESYSTEM_PATH` to the path where the files will be stored.

### FFMPEG

If the `ffmpeg` and `ffprobe` binaries are not in `/usr/bin`, you must specify its location:

| Variable Name | Description              |
| ------------- | ------------------------ |
| FFMPEG_PATH   | Path to `ffmpeg` binary  |
| FFPROBE_PATH  | Path to `ffprobe` binary |

### More options

Here is a list with more options you can configure:

| Variable Name | Description                               |
| ------------- | ----------------------------------------- |
| LOG_REQUESTS  | Set to `YES` or `NO`. By default is `YES` |
| LOG_DEBUG     | Set to `YES` or `NO`. By default is `NO`  |
