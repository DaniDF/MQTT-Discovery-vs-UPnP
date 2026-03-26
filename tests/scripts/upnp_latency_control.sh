#!/bin/bash

REMOTE_ROOT='root@10.2.0.107'
REMOTE_USER='user@10.2.0.107'

## Test latency

trap ctrl_c INT

function kill_remote_jobs() {
    for pid in $(ssh $REMOTE_ROOT 'ps aux | grep -E "[0-9]+ (bash -c )?tests/bin/device" | awk '\''{$1=$1}1'\'' | cut -d" " -f2') ;
    do
        ssh $REMOTE_ROOT 'kill -9 '$pid
    done
}

function ctrl_c() {
    echo STOPPING...
    kill_remote_jobs
    ssh -O exit $REMOTE_USER > /dev/null
    exit
}

### Soap test
for upnp_devices in {1..9} {10..100..10} ;
do
    for upnp_controls in {1..9} {10..100..10} ;
    do
        echo "Running upnp_devices: $upnp_devices, upnp_controls $upnp_controls"

        for count in {1..30} ;
        do
            ssh -f $REMOTE_USER "tests/bin/device -u $upnp_devices > tests/logs/device_test-latency_u-"$upnp_devices"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null"
            
            sleep 3

            ../bin/./control -u $upnp_controls > ../logs/control_test-latency_d-"$upnp_devices"_u-"$upnp_controls"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null
        
            kill_remote_jobs
            sleep 1
        done
    done
done

### Gena test

for upnp_controls in {1..9} {10..100..10} ;
do
    echo "Running upnp_controls $upnp_controls"
    
    for count in {1..30} ;
    do
        ssh $REMOTE_USER "tests/bin/device -u 1 > tests/logs/device_test-gena_u-1_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null"
        
        sleep 3
        ../bin/./control -u $upnp_controls > ../logs/control_test-gena_d-1_u-"$upnp_controls"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null

        kill_remote_jobs
        sleep 1
    done
done