#!/bin/bash 
set +x
trap ctrl_c INT

function ctrl_c() {    
    echo "** Trapped CTRL-C. Stopping  processes..."
    killProcesses
}

function killProcesses() {
    for pid in ${pids[*]}; do
        echo -n "Stopping local PID: $pid"
        kill $pid
        echo " stopped."
    done

    for pid in ${rempids[*]}; do
        echo -n "Stopping remote PID: $pid"
        ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP sudo kill $pid
        echo " stopped."
    done
    
    echo "All Stopped."
    exit 1
}

BASE=$( git rev-parse --show-toplevel)
SOCKETFILE="$BASE/cmd/aro/mdm_statsd.socket"
CLOUDUSER="cloud-user"

echo "Using:"

echo "Resourcegroup = $RESOURCEGROUP"
echo "MDM_VM_NAME   = mdm-$LOCATION"

echo "Looking for a VM called $MDM_VM_NAME and its Private IP"
VM_IP_PROPERTY=".[0].virtualMachine.network.publicIpAddresses[0].ipAddress"
if [ "$MDM_VM_PRIVATE" != "" ] || [ "$MDM_VM_PRIVATE" == "null" ];
then
    VM_IP_PROPERTY=".[0].virtualMachine.network.privateIpAddresses[0]"
fi   
MDM_VM_IP=$( az vm list-ip-addresses --name $MDM_VM_NAME -g $RESOURCEGROUP -o json | jq -r $VM_IP_PROPERTY )

if [ "$MDM_VM_IP" == "" ] || [ "$MDM_VM_IP" == "null" ]; then
    echo "ERR: no IP address found for $MDM_VM_NAME. Giving up."
    exit 2
fi

echo -n "Found IP $MDM_VM_IP, starting socat on the mdm-link vm"
ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP 'sudo socat -v TCP-LISTEN:12345,fork UNIX-CONNECT:/var/etw/mdm_statsd.socket'  &
sleep 2
REMPS=$( ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP 'ps aux | grep "socat -v TCP-LISTEN:12345,fork UNIX-CONNECT:/var/etw/mdm_statsd.socket" | grep -v sudo | grep -v grep' )
REMPID=$( echo $REMPS |  awk '{print $2}' )
if [ "$REMPID" == "" ]; then
    echo ""
    echo "ERR: FAILED TO START REMOTE SOCAT.."
    killProcesses
    exit 1
fi

rempids[0]=$REMPID
echo "...remote socat started."
echo -n "Starting SSH Tunnel..." 
ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP -N -L 12345:127.0.0.1:12345 -o ConnectTimeout=4 &
pids[0]=$!
sleep 5
kill -0 ${pids[0]}
if [ $? -eq 0 ]; then
    echo "...SSH Tunnel started. PID: ${pids[0]}."
else
    echo ""
    echo "ERR: FAILED TO START TUNNEL.."
    killProcesses
    exit 1
fi

if [ -e "$SOCKETFILE" ] ; then
    echo "Cleaning up old socket file."
    rm "$SOCKETFILE"
fi
echo -n "Starting local socat link to the tunnel..."
socat -v UNIX-LISTEN:$SOCKETFILE,fork TCP-CONNECT:127.0.0.1:12345 &
pids[1]=$!
sleep 2

kill -0 ${pids[1]}
if [ $? -eq 0 ]; then
    echo "...local socat started.PID: ${pids[1]}."
else
    echo "ERR: FAILED TO START SOCAT. Cleaning up:"    
    echo "Killing SSH tunnel"
    killProcesses
    echo "Killed."
    exit 1
fi

echo ""
echo "**********************************************************************"
echo "*  Remote socat: Started. SSH Tunnel: Started. Local socat: Started. *"
echo "*                                                                    *"
echo "*      Hit CTRL-C to stop                                            *"
echo "*                                                                    *"
echo "**********************************************************************"
echo ""
echo ""
while true
do
    sleep 100
done
