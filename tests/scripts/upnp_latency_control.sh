#!/bin/bash

## Test latency

trap ctrl_c INT

function ctrl_c() {
    echo STOPPING...
    ssh -S /tmp/ssh-device -O exit user@10.2.0.107 > /dev/null
    exit
}

for upnp_devices in {1..9} {10..100..10} ;
do
    ssh -f -MS /tmp/ssh-device user@10.2.0.107 "tests/bin/device -u $upnp_devices > tests/logs/device_test-latency_u-"$upnp_devices"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null"

    sleep 3

    for upnp_controls in {1..9} {10..100..10} ;
    do
        echo "Running upnp_devices: $upnp_devices, upnp_controls $upnp_controls"
        
        for count in {1..30} ;
        do
            sleep 2
            ../bin/./control -u $upnp_controls > ../logs/control_test-latency_d-"$upnp_devices"_u-"$upnp_controls"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null
        done
    done

    ssh root@10.2.0.107 'pkill -f "tests/bin/device -u"' > /dev/null
    ssh -S /tmp/ssh-device -O exit user@10.2.0.107 > /dev/null 2>&1
done