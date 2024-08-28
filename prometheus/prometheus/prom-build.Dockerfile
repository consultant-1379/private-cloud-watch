FROM ubuntu:16.04

ARG GO_TAR
ARG USER
ARG UID
ARG GROUP
ARG GID

RUN apt-get update && \
    apt-get install -y \
	build-essential \
	curl

RUN rm -rf /var/cache/apt/* /var/lib/apt/lists/*

ADD $GO_TAR /usr/local

RUN groupadd -r -g $GID $GROUP || true
RUN useradd --no-log-init -r -g $GROUP -u $UID $USER || true

RUN mkdir /build && chown $UID:$GID /build
RUN mkdir /src && chown $UID:$GID /src
USER $UID:$GID

WORKDIR /build
ENV GOROOT /usr/local/go
ENV GOPATH /build
ENV USER $USER
ENV HOME /build
ENV PATH $GOPATH/bin:$GOROOT/bin:$PATH
