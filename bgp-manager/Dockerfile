FROM debian:10

ARG DEBIAN_FRONTEND="noninteractive"
RUN apt update && \
    apt install -y gnupg2 \
                   curl \
                   apt-transport-https \
                   procps

RUN curl -s https://deb.frrouting.org/frr/keys.asc | apt-key add -
RUN echo deb https://deb.frrouting.org/frr stretch "frr-stable" | tee -a /etc/apt/sources.list.d/frr.list
RUN apt update && \
    apt install -y frr
RUN rm /etc/frr/frr.conf
COPY frr/ /etc/frr

COPY cni-bin/megalos-amd64 /megalos-amd64
COPY cni-bin/megalos-arm64 /megalos-arm64

RUN apt clean && \
    rm -rf /tmp/* /var/lib/apt/lists/* /var/tmp/*

COPY ./entrypoint.sh /entrypoint.sh
RUN chmod u+x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]