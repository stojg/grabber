all:
	docker build -t grabber .
	docker tag grabber:latest 112402665134.dkr.ecr.ap-southeast-2.amazonaws.com/grabber:latest
	docker push 112402665134.dkr.ecr.ap-southeast-2.amazonaws.com/grabber:latest

login:
	aws ecr get-login --no-include-email --region ap-southeast-2
