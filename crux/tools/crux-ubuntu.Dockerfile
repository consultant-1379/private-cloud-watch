FROM ubuntu:16.04

RUN apt-get update && \
    apt-get install -y \
	apt-utils \
	dnsutils \
	python-dev \
	python-pip \
	ssh \
	syslog-ng \
	syslog-ng-core \
	telnet \
	unzip
