ARG GO_VERSION=1.26
ARG ALPINE_VERSION=3.23

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder
ARG ARCH
ARG OS
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN GOOS=$OS GOARCH=$ARCH go build -o app .

FROM alpine
RUN mkdir /app
WORKDIR /app
COPY --from=builder /app/app .
CMD ["./app"]
