FROM ubuntu:16.04

ARG USER
ARG UID
ARG GROUP
ARG GID

RUN rm -rf /var/cache/apt/* /var/lib/apt/lists/*

RUN groupadd -r -g $GID $GROUP || true
RUN useradd --no-log-init -r -g $GROUP -u $UID $USER || true

RUN mkdir -p /usr/local/bin
ADD prom-bin.tar.gz /usr/local/bin
COPY exec-prometheus.sh /usr/local/bin
RUN mkdir /prometheus && chown $UID:$GID /prometheus
USER $UID:$GID

WORKDIR /prometheus
ENV USER $USER
ENV HOME /prometheus
ENTRYPOINT ["/usr/local/bin/exec-prometheus.sh"]

## docker run --detach --rm --cidfile ./prometheus.cid -p 9090:9090 -v $(pwd):/prometheus erixzone/prom-run prometheus.log --config.file prometheus.yaml
## docker container kill --signal=HUP `cat ./prometheus.cid` # reload config
## docker container kill --signal=INT `cat ./prometheus.cid` && rm ./prometheus.cid
