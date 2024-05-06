# Streaming files structure

The video streams will be encoded and saved into a shared file system between the encoder and the application-specific servers.

## HLS (HTTP Live streaming)

The video streams are encoded into the HLS format. This format consists of two kind of files:

 - The video fragments, with `.ts` extension.
 - The playlists, with `.m3u8` extension.

Each video stream is encoded in multiple resolutions, each resolution having the format `{WIDTH}x{HEIGHT}-{FPS}`. Example: `1280x720-30`.

For each resolution, the following files are stored:

 - HLS fragments: Each fragment has a number, increasing in order. Path pattern: `hls/{Stream-Channel}/{Stream-ID}/{WIDTH}x{HEIGHT}-{FPS}~{BITRATE}/{Fragment-Number}.ts`
 - HLS live playlist: Playlist to fetch the stream while it's being broadcasted. Path pattern: `hls/{Stream-Channel}/{Stream-ID}/{WIDTH}x{HEIGHT}-{FPS}~{BITRATE}/live.m3u8`
 - HLS VOD playlist: Playlist to fetch the stream as a video on demand. Path pattern: `hls/{Stream-Channel}/{Stream-ID}/{WIDTH}x{HEIGHT}-{FPS}~{BITRATE}/vod-{VOD-INDEX}.m3u8`

## Stream preview images

If enabled, the encoders can generate snapshot images of the video stream each fixed number of seconds.

The images are stored in the JPEG format, with the `.jpg` extension.

Each image has an index number, increasing in one for each one.

The pattern for images is the following: `img-preview/{Stream-Channel}/{Stream-ID}/{WIDTH}x{HEIGHT}-{SECONDS}/{Image-Index}.jpg`

Also, a JSON index file is stored, with pattern `img-preview/{Stream-Channel}/{Stream-ID}/{WIDTH}x{HEIGHT}-{SECONDS}/index.json` containing an object with the following properties:

 - `index_start` - Index number of the first available image.
 - `count` - Total number of available images
 - `pattern` - Image file pattern. `%d` is replaced with the image index.

Example:

```json
{
    "index_start": 0,
    "count": 100,
    "pattern": "%d.jpg"
}
```
