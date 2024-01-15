# -------------
# Build stage

FROM golang:1.21 AS builder

RUN mkdir -p /greta

RUN groupadd -g 1000 ec2-user && useradd -r -s /bin/false -u 1000 -g ec2-user ec2-user

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
WORKDIR /home/ec2-user/greta
COPY --from=builder /greta/health /home/ec2-user/greta/health
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

RUN mkdir /home/ec2-user/greta/healthtests && chown -R ec2-user:ec2-user /home/ec2-user/greta/
VOLUME ["/home/ec2-user/greta/healthtests"]

USER ec2-user
