FROM golang:alpine

WORKDIR /root

# Copy backend

ADD backend /root/backend

# Copy frontend

ADD frontend/dist /root/frontend

ENV FRONTEND_PATH=/root/frontend

# Compile backend

WORKDIR /root/backend

RUN go build -o backend

WORKDIR /root

# Ports

EXPOSE 80

# Entry point

ENTRYPOINT ["/root/backend/backend"]
