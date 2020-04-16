LDFLAGS := -ldflags='-s -w -X "main.Version=$(VERSION)"'
GIT_COMMIT := $(shell git rev-parse HEAD 2> /dev/null || true)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null)

all:
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -trimpath ${LDFLAGS} -o app .
	tar cfz zoneinfo.tar.gz /usr/share/zoneinfo
	docker build -t stojg/grabber:$(GIT_COMMIT) .
	rm zoneinfo.tar.gz
	rm app

build:
	go build ${LDFLAGS}

compose:
	docker-compose build
	docker-compose up

clean:
	docker-compose rm --stop  --force -v
	docker volume rm grabber_chronograf-storage grabber_grafana-storage grabber_influxdb-storage

release: all
	docker tag stojg/grabber:$(GIT_COMMIT) stojg/grabber:latest
	docker push stojg/grabber:$(GIT_COMMIT)
