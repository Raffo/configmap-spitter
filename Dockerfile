FROM golang:1.13 as builder

COPY . /go/src/github.com/Raffo/configmaps-to-volume

RUN cd /go/src/github.com/Raffo/configmaps-to-volume && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build .

FROM alpine:3.9

USER nobody

COPY --from=builder /go/src/github.com/Raffo/configmaps-to-volume/configmaps-to-volume /configmaps-to-volume

RUN ls -la
# COPY build/configmaps-to-volume-linux-amd64 /configmaps-to-volume

ENTRYPOINT ["/configmaps-to-volume"]