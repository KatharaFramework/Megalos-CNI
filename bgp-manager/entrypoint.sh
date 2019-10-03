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

# Run the BGP Manager
python /mgr/app.py