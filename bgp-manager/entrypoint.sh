#!/bin/sh

# Always exit on errors.
set -e

if [ "$IS_MASTER" = "false" ]; then
  CNI_BIN_DIR="/host/opt/cni/bin"
  KATHARA_CNI_BIN_FILE="megalos"

  # Copy CNI Plugin File into host's /opt/cni/bin
  cp -f /$KATHARA_CNI_BIN_FILE $CNI_BIN_DIR/_megalos
  mv -f $CNI_BIN_DIR/_megalos $CNI_BIN_DIR/megalos

  # Configure frr
  rm -Rf /etc/frr/bgpd_master.stub
  mv /etc/frr/bgpd_worker.stub /etc/frr/frr.conf
  sed -i -e "s|__NODE_IP__|$NODE_IP|g" /etc/frr/frr.conf
  while [ -z "$KATHARA_MASTER_SERVICE_HOST" ]; do sleep 1; done;
  sed -i -e "s|__SERVICE_IP__|$KATHARA_MASTER_SERVICE_HOST|g" /etc/frr/frr.conf
else
  # Configure frr
  rm -Rf /etc/frr/bgpd_worker.stub
  mv /etc/frr/bgpd_master.stub /etc/frr/frr.conf
fi

# Start frr
/etc/init.d/frr start

sleep infinity