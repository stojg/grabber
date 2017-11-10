FROM       scratch
MAINTAINER Stig Lindqvist <stig@stojg.se>
ADD zoneinfo.tar.gz /
ADD        app app
ENTRYPOINT ["/app"]
