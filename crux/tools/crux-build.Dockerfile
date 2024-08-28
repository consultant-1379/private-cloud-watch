# Pre-build container
FROM erixzone/crux-ubuntu

ARG go_version=1.12.6
ARG protoc_version=3.3.0
ARG go_url=https://dl.google.com/go/go${go_version}.linux-amd64.tar.gz
ARG protoc_url=https://github.com/google/protobuf/releases/download/v${protoc_version}/protoc-${protoc_version}-linux-x86_64.zip

ENV GOROOT /usr/local/go/

RUN apt-get update && \
    apt-get install -y \
	autoconf \
	build-essential \
	ca-certificates \
	curl \
	gawk \
	git \
	libtool \
	pkg-config \
	unzip \
	wget \
	yasm && \
    rm -r /var/cache/apt/* /var/lib/apt/lists/*

RUN wget --quiet ${go_url} && \
    tar -xzf go*.tar.gz -C ${GOROOT%*go*} && \
    rm go*.tar.gz && \
    ln -s ${GOROOT}/bin/* /usr/local/bin/

RUN CURDIR=$(pwd) ; wget --quiet ${protoc_url} && \
    cd /usr/local && \
    unzip "${CURDIR}"/protoc*.zip && \
    chmod -R a+rx bin/protoc include/google && \
    rm readme.txt && \
    cd "${CURDIR}" && \
    rm protoc*.zip

WORKDIR /crux/src/github.com/erixzone/crux
ENV GOPATH /crux
ENV PATH $GOPATH/bin:$PATH
