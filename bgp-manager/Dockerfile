FROM ubuntu:16.04

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    wget \
    gnupg \
    lsb-release \
    python-pip

RUN wget -O- https://apps3.cumulusnetworks.com/setup/cumulus-apps-deb.pubkey | apt-key add -   
   
RUN echo "deb [arch=amd64] http://apps3.cumulusnetworks.com/repos/deb $(lsb_release -cs) roh-3" >> /etc/apt/sources.list.d/cumulus-apps-deb-$(lsb_release -cs).list

RUN apt-get update && apt-get install -y frr
RUN ln -s /usr/lib/frr/frr /etc/init.d/frr

COPY cni-bin/megalos /megalos

COPY frr/ /etc/frr

COPY ./entrypoint.sh /entrypoint.sh
RUN chmod u+x /entrypoint.sh

COPY mgr/ /mgr
RUN pip install -r /mgr/requirements.txt

ENTRYPOINT ["/entrypoint.sh"]