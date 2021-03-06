FROM golang:1.13-alpine as golang
WORKDIR /go/src/app/vendor/github.com/stojg/grabber
COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o /go/bin/app

FROM alpine:latest as alpine
RUN apk --no-cache add tzdata zip ca-certificates
WORKDIR /usr/share/zoneinfo
RUN zip -r -0 /zoneinfo.zip .

FROM scratch
ENV ZONEINFO /zoneinfo.zip
COPY --from=alpine /zoneinfo.zip /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=golang /go/bin/app /app
ENTRYPOINT ["/app"]
