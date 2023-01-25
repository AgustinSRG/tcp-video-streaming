FROM golang:latest

WORKDIR /root

# Copy files

ADD . /root

# Fetch dependencies

RUN go get .

# Compile

RUN go build -o ws-stream-server

# Expose ports

EXPOSE 80
EXPOSE 443

# Entry point

ENTRYPOINT ["/root/ws-stream-server"]
