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
| CONTROL_BASE_URL | Websocket URL to connect to the coordinator server. Example: `wss://10.0.0.0:8080/`                |
| CONTROL_SECRET   | Secret shared between the coordinator server and the HLS encoder server, in order to authenticate. |

### Storage

You can configure the storage system where the HLS files will be stored by changing the variable `HLS_STORAGE_TYPE`.

| Variable Name    | Description                              |
| ---------------- | ---------------------------------------- |
| HLS_STORAGE_TYPE | HLS Storage type. Default: `FILESYSTEM`. |

### Storage (File system)

Set `HLS_STORAGE_TYPE` to `FILESYSTEM` in order to store the files in a folder specified by `HLS_FILESYSTEM_PATH`.

| Variable Name       | Description                                                                             |
| ------------------- | --------------------------------------------------------------------------------------- |
| HLS_FILESYSTEM_PATH | Path where the HLS files will be stored. It may be a remote or distributed file system. |

### Storage (HTTP)

Set `HLS_STORAGE_TYPE` to `HTTP` in order to send HTTP requests to store the files.

The request method is either `PUT` (to save a file) or `DELETE` (to delete a file).

The request path is the route of the file.

The request body is the contents of the file (only for `PUT` requests).

| Variable Name                | Description                                                                                                   |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------- |
| HLS_STORAGE_HTTP_URL         | Base URL to send the HTTP requests.                                                                           |
| HLS_STORAGE_HTTP_AUTH        | Authorization type for HTTP request. Leave empty for no authentication. Can be: `Basic`, `Bearer` or `Custom` |
| HLS_STORAGE_HTTP_USER        | Authorization user (`Basic`)                                                                                  |
| HLS_STORAGE_HTTP_PASSWORD    | Authorization password (`Basic`)                                                                              |
| HLS_STORAGE_HTTP_TOKEN       | Authorization bearer token (`Bearer`)                                                                         |
| HLS_STORAGE_HTTP_AUTH_CUSTOM | Custom value of the `Authorization` header (`Custom`)                                                         |

### Storage (AWS S3)

Set `HLS_STORAGE_TYPE` to `S3` in order to store files in AWS S3.

| Variable Name         | Description                |
| --------------------- | -------------------------- |
| AWS_REGION            | Name of the AWS region.    |
| AWS_S3_BUCKET         | Name of the AWS S3 bucket. |
| AWS_ACCESS_KEY_ID     | AWS access key ID.         |
| AWS_SECRET_ACCESS_KEY | AWS secret access key.     |

### Storage (Azure Blob Storage)

Set `HLS_STORAGE_TYPE` to `AZ` in order to store files in Azure Blob Storage

| Variable Name           | Description                    |
| ----------------------- | ------------------------------ |
| AZURE_STORAGE_ACCOUNT   | ID of the storage account.     |
| AZURE_STORAGE_CONTAINER | Name of the storage container. |
| AZURE_TENANT_ID         | Azure tenant ID.               |
| AZURE_CLIENT_ID         | Azure client ID.               |
| AZURE_CLIENT_SECRET     | Azure client secret.           |

### HLS websocket CDN

Settings to publish the live HLS streams to [HLS Websocket CDN](https://github.com/AgustinSRG/hls-websocket-cdn).

| Variable Name          | Description                                                                                       |
| ---------------------- | ------------------------------------------------------------------------------------------------- |
| HLS_WS_CDN_ENABLED     | Set to `YES` or `NO`. Set it to `YES` if the encoder must publish the stream to the CDN.          |
| HLS_WS_CDN_URL         | Websocket URL of the CDN server. You can specify multiple servers, separating the URLs by spaces. |
| HLS_WS_CDN_PUSH_SECRET | Secret to authenticate the PUSH requests to the CDN server.                                       |

The stream IDs for the CDN will be the index files. Example:

```
hls/channel-id/stream-id/800x600-30~1000/live.m3u8
```

### FFMPEG

If the `ffmpeg` and `ffprobe` binaries are not in `/usr/bin`, you must specify its location:

| Variable Name | Description              |
| ------------- | ------------------------ |
| FFMPEG_PATH   | Path to `ffmpeg` binary  |
| FFPROBE_PATH  | Path to `ffprobe` binary |

### HLS

Additional configuration for HLS

| Variable Name            | Description                                                                                                                            |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------- |
| HLS_VIDEO_CODEC          | Video codec to use to encode HLS fragments. Default: `libx264`. Check [FFmpeg codecs](https://www.ffmpeg.org/ffmpeg-codecs.html).      |
| HLS_AUDIO_CODEC          | Audio codec to use to encode HLS fragments. Default: `aac`. Check [FFmpeg codecs](https://www.ffmpeg.org/ffmpeg-codecs.html).          |
| HLS_TIME_SECONDS         | Duration (seconds) of each video fragment (by default 3 seconds).                                                                      |
| HLS_LIVE_PLAYLIST_SIZE   | Max number of fragments in the live playlist (10 by default)                                                                           |
| HLS_VOD_MAX_SIZE         | Max number of fragments to include in a single VOD playlist. Default value: `86400`                                                    |
| HLS_FRAGMENT_COUNT_LIMIT | Max number of fragments to allow in a single stream. After this limit is reached, the stream will be closed. Default value: `16777216` |

### More options

Here is a list with more options you can configure:

| Variable Name   | Description                               |
| --------------- | ----------------------------------------- |
| LOG_TASK_STATUS | Set to `YES` or `NO`. By default is `YES` |
| LOG_DEBUG       | Set to `YES` or `NO`. By default is `NO`  |
