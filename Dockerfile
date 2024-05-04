# Build the tools

FROM golang:latest

WORKDIR /root

# Copy files

ADD . /root

RUN mkdir /root/dist

# Build: coordinator

WORKDIR /root/coordinator

RUN go build .

RUN cp coordinator /root/dist/coordinator

# Build: coordinator

WORKDIR /root/hls-encoder

RUN go build .

RUN cp hls-encoder /root/dist/hls-encoder

# Build: rtmp-server

WORKDIR /root/rtmp-server

RUN go build .

RUN cp rtmp-server /root/dist/rtmp-server

# Build: ws-stream-server

WORKDIR /root/ws-stream-server

RUN go build .

RUN cp ws-stream-server /root/dist/ws-stream-server

# Prepare runner

FROM alpine as runner

# Add gcompat

RUN apk add gcompat

# Add FFmpeg

RUN apk add --no-cache ffmpeg
ENV FFMPEG_PATH=/usr/bin/ffmpeg
ENV FFPROBE_PATH=/usr/bin/ffprobe

# Copy binaries

COPY --from=0 /root/dist /root/dist

# Expose ports

ENV HTTP_PORT=80
ENV SSL_PORT=443
ENV RTMP_PORT=1935

EXPOSE 80
EXPOSE 443
EXPOSE 1935

# Entry point

WORKDIR /root/dist

ENV PATH="${PATH}:/root/dist"

ENTRYPOINT ["sh", "-c"]
