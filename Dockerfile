FROM golang:1.25-alpine AS builder

ARG VERSION=dev

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X 'github.com/jsimonovski/helmver/cmd.version=${VERSION}'" \
    -o /helmver .

FROM alpine:3.20

RUN apk add --no-cache git

COPY --from=builder /helmver /usr/local/bin/helmver

ENTRYPOINT ["helmver"]
