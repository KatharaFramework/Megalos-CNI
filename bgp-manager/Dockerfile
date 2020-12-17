FROM debian:9

RUN apt update && DEBIAN_FRONTEND=noninteractive apt install -y python3 python3-pip curl apt-transport-https

RUN curl -s https://deb.frrouting.org/frr/keys.asc | apt-key add -

RUN echo deb https://deb.frrouting.org/frr stretch "frr-stable" | tee -a /etc/apt/sources.list.d/frr.list

RUN apt update && DEBIAN_FRONTEND=noninteractive apt install -y frr

RUN rm /etc/frr/frr.conf

RUN apt clean

COPY cni-bin/megalos /megalos

COPY frr/ /etc/frr

COPY ./entrypoint.sh /entrypoint.sh
RUN chmod u+x /entrypoint.sh

COPY mgr/ /mgr
RUN python3 -m pip install -r /mgr/requirements.txt

ENTRYPOINT ["/entrypoint.sh"]
