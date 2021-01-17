# -------------
# Build stage

FROM golang:1.15 AS builder

RUN mkdir -p /greta

RUN groupadd -g 1000 greta && useradd -r -s /bin/false -u 1000 -g greta greta

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
WORKDIR /greta
COPY --from=builder /greta/health /greta/health
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

USER 1000
COPY scripts /greta/scripts
