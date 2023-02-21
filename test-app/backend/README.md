# Streaming test backend server

This is a simple backend server to test the TCP video streaming tools.

Note: Do not use for production. This is only meant for local testing.

## Compilation

In order to install dependencies, type:

```
go get .
```

To compile the code type:

```
go build
```

The build command will create a binary in the current directory, called `backend`, or `backend.exe` if you are using Windows.

## Usage

In order to run the server, run the binary without arguments.

```
./backend
```

## Configuration

You can configure the server with environment variables.

| Variable Name              | Description                                                                                   |
| -------------------------- | --------------------------------------------------------------------------------------------- |
| HTTP_PORT                  | HTTP listening port. Default is `80`                                                          |
| BIND_ADDRESS               | Bind address for RTMP and RTMPS. By default it binds to all network interfaces.               |
| FRONTEND_LOCATION          | Path to the frontend, in order to serve it.                                                   |
| CONTROL_SERVER_BASE_URL    | Base URL of the control server, to send commands. Example: `http://localhost:8080`            |
| DB_PATH                    | Path to store the JSON database. By default it will store it in the current working directory |
| CORS_INSECURE_MODE_ENABLED | Set it to `YES` to allow insecure CORS requests (for development purposes)                    |

### Key verification

This test app provides a key verification callback in the path `/callbacks/key_verification`

In order to test the different kinds of authentication, you can set the following options (Set for the `KEY_VERIFICATION_AUTH` environment variable):

 - `Basic` - Basic HTTP authorization. Set `KEY_VERIFICATION_AUTH_USER` and `KEY_VERIFICATION_PASSWORD` environment variables.
 - `Bearer` - Bearer token authentication. Set `KEY_VERIFICATION_AUTH_TOKEN` environment variable.
 - `Custom` - Custom authentication header. Set `KEY_VERIFICATION_AUTH_CUSTOM` environment variable.
