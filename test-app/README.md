# Streaming test application

This folder contains a simple test application for the TCP video streaming tools.

This test app provides a simple frontend that allows you to generate streaming keys and playback the active streams using HLS in the browser.

## Running with Docker

You need Docker and Docker-Compose installed. 

First, set up a `.env` file in this path, in order to configure the following environment variables:

| Variable        | Description                                                                                         |
| --------------- | --------------------------------------------------------------------------------------------------- |
| `SHARED_SECRET` | Secret to share between the test application and the streaming components. Must be a random string. |

Then, you can simply run the test application by typing:

```
docker compose up -d
```

You can access the test application at http://localhost

You can stop it using:

```
docker compose down
```
