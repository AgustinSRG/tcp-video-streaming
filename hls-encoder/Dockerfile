FROM golang:latest

# Dependencies

RUN apk add --no-cache ffmpeg
ENV FFMPEG_PATH=/usr/bin/ffmpeg
ENV FFPROBE_PATH=/usr/bin/ffprobe

WORKDIR /root

# Copy files

ADD . /root

# Fetch dependencies

RUN go get .

# Compile

RUN go build -o hls-encoder

# Entry point

ENTRYPOINT ["/root/hls-encoder"]
