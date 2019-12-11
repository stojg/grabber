LDFLAGS := -ldflags='-s -w -X "main.Version=$(VERSION)"'

all:
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -trimpath ${LDFLAGS} -o app .
	tar cfz zoneinfo.tar.gz /usr/share/zoneinfo
	docker build -t stojg/grabber .
	rm zoneinfo.tar.gz
	rm app


build:
	go build ${LDFLAGS}

compose:
	docker-compose up

setup:
	curl -i -XPOST http://localhost:8086/query --data-urlencode "q=CREATE DATABASE grabber"

clean:
	docker volume rm grabber_chronograf-storage grabber_grafana-storage grabber_influxdb-storage
