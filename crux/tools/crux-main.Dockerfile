FROM erixzone/crux-ubuntu

RUN rm -r /var/cache/apt/* /var/lib/apt/lists/*

COPY crux-main.syslog-ng.conf /usr/
RUN cat /usr/crux-main.syslog-ng.conf >> /etc/syslog-ng/syslog-ng.conf

RUN groupadd -r crux
RUN useradd --no-log-init -r -g crux crux

RUN mkdir /crux /crux/bin; chown crux:crux /crux /crux/bin; chmod 700 /crux /crux/bin
ADD crux.tar.gz /crux

USER crux
WORKDIR /crux
ENV PATH="/crux/bin:${PATH}"
