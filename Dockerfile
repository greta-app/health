# -------------
# Build stage

FROM golang:1.23.4-alpine3.21 AS builder

RUN mkdir -p /greta

WORKDIR /greta
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo \
        -o /greta/health \
        main.go

# -------------
# Image creation stage

FROM alpine:latest
RUN apk add curl bash
RUN addgroup -g 1000 -S ec2-user && adduser -S -D -u 1000 -G ec2-user ec2-user
WORKDIR /home/ec2-user/greta
COPY --from=builder /greta/health /home/ec2-user/greta/health

RUN mkdir /home/ec2-user/greta/healthtests && chown -R ec2-user:ec2-user /home/ec2-user/greta/
VOLUME ["/home/ec2-user/greta/healthtests"]

USER ec2-user
