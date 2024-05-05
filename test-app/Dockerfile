# Step 1: Backend

FROM golang:latest

WORKDIR /root

# Copy backend

ADD backend /root/backend

# Compile backend

WORKDIR /root/backend

RUN go build .

# Step 2: Frontend

FROM node:20-alpine

WORKDIR /root

RUN mkdir /root/frontend

WORKDIR /root/frontend

ADD frontend/package.json /root/frontend/package.json
ADD frontend/package-lock.json /root/frontend/package-lock.json

RUN npm install

ADD frontend /root/frontend

RUN npm run build

# Step 3: Prepare runner

FROM alpine as runner

# Add gcompat

RUN apk add gcompat

# Copy files

COPY --from=0 /root/backend/backend /root/backend
COPY --from=1 /root/frontend/dist /root/frontend

# Env vars

ENV FRONTEND_PATH=/root/frontend

# Ports

EXPOSE 80

# Entry point

ENTRYPOINT ["/root/backend"]
