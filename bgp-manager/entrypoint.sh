#!/bin/sh

# Always exit on errors.
set -e

if [ "$IS_MASTER" = "false" ]; then
    CNI_BIN_DIR="/host/opt/cni/bin"
    KATHARA_CNI_BIN_FILE="megalos"

    # Copy CNI Plugin File into host's /opt/cni/bin
    cp -f /$KATHARA_CNI_BIN_FILE $CNI_BIN_DIR/_megalos
    mv -f $CNI_BIN_DIR/_megalos $CNI_BIN_DIR/megalos
fi

# sysctl -w net.core.rmem_max=16777216
# sysctl -w net.core.wmem_max=16777216

# Run the BGP Manager
python3 /mgr/app.py

sleep infinity