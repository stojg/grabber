all:
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o app .
	tar cfz zoneinfo.tar.gz /usr/share/zoneinfo
	docker build -t stojg/grabber .
	rm zoneinfo.tar.gz
	rm app

login:
	aws ecr get-login --no-include-email --region ap-southeast-2

push:
	docker tag stojg/grabber:latest 112402665134.dkr.ecr.ap-southeast-2.amazonaws.com/grabber:latest
	docker push 112402665134.dkr.ecr.ap-southeast-2.amazonaws.com/grabber:latest
