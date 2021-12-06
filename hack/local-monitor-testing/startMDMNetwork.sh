#!/bin/bash -e
set +x

BASE=$( git rev-parse --show-toplevel)
SOCKETFILE="$BASE/cmd/aro/mdm_statsd.socket"

echo "Using:"

echo "Resourcegroup = $RESOURCEGROUP"
echo "User          = $USER"

VMName="$USER-mdm-link"
CLOUDUSER="cloud-user"


PUBLICIP=$( az vm list-ip-addresses --name $VMName -g $RESOURCEGROUP | jq -r '.[0].virtualMachine.network.publicIpAddresses[0].ipAddress' )

echo "Found IP $PUBLICIP, starting socat on the mdm-link vm"
ssh $CLOUDUSER@$PUBLICIP "sudo socat -v TCP-LISTEN:12345,fork UNIX-CONNECT:/var/etw/mdm_statsd.socket" &
sleep 3

echo "Starting SSH Tunnel"
ssh $CLOUDUSER@$PUBLICIP -N -L 12345:127.0.0.1:12345  &
sleep 3

if [ -f "$SOCKETFILE" ] ; then
    rm "$SOCKETFILE"
fi
echo "Starting local socat link to the tunnel"
socat -v UNIX-LISTEN:$SOCKETFILE,fork TCP-CONNECT:127.0.0.1:12345 &


