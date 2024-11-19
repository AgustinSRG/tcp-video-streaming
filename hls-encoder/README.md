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

- `FILESYSTEM` - Store the files in a folder specified in `HLS_FILESYSTEM_PATH`
- `HTTP` - Send HTTP requests to store the files. The request method is either `PUT` or `DELETE` The request path is the route of the file. The request body is the contents of the file.
- `S3` - Store files in AWS S3
- `AZURE_BLOB_STORAGE` - Store files in Azure Blob Storage

| Variable Name           | Description                                                                             |
| ----------------------- | --------------------------------------------------------------------------------------- |
| HLS_STORAGE_TYPE        | HLS Storage type. Default: `FILESYSTEM`.                                                |
| HLS_FILESYSTEM_PATH     | Path where the HLS files will be stored. It may be a remote or distributed file system. |
| HLS_STORAGE_HTTP_URL    | Base URL to send the PUT requests.                                                      |
| AWS_REGION              | Name of the AWS region (when storing files in AWS S3).                                  |
| AWS_S3_BUCKET           | Name of the AWS S3 bucket (when storing files in AWS S3).                               |
| AWS_ACCESS_KEY_ID       | AWS access key ID (when storing files in AWS S3).                                       |
| AWS_SECRET_ACCESS_KEY   | AWS secret access key (when storing files in AWS S3).                                   |
| AZURE_STORAGE_ACCOUNT   | ID of the storage account (when storing files in Azure Blob Storage).                   |
| AZURE_STORAGE_CONTAINER | Name of the storage container (when storing files in Azure Blob Storage).               |
| AZURE_TENANT_ID         | Azure tenant ID (when storing files in Azure Blob Storage).                             |
| AZURE_CLIENT_ID         | Azure client ID (when storing files in Azure Blob Storage).                             |
| AZURE_CLIENT_SECRET     | Azure client secret (when storing files in Azure Blob Storage).                         |

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
