FROM golang:latest

WORKDIR /root

# Copy files

ADD . /root

# Fetch dependencies

RUN go get .

# Compile

RUN go build -o rtmp-server

# Expose ports

EXPOSE 1935
EXPOSE 443

# Entry point

ENTRYPOINT ["/root/rtmp-server"]
