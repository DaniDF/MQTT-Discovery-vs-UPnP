#!/bin/bash

## Test latency

for upnp_devices in {1..100} ;
do
    ssh -f -MS /tmp/ssh-device user@10.2.0.107 "tests/bin/device -u $upnp_devices --mqtt-broker 10.2.0.108:1883 -m 0 --qos 0 > tests/logs/device_test-latency_u-"$upnp_devices"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null"
    
    for upnp_controls in {1..100} ;
    do
        echo "Running upnp_devices: $upnp_devices, upnp_controls $upnp_controls"
        
        for count in {1..50} ;
        do
            ../bin/./control -u "$upnp_controls" --mqtt-broker 10.2.0.108:1883 -m 0 --qos 0 > ../logs/control_test-latency_d-"$upnp_devices"_u-"$upnp_controls"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null
        done
    done

    ssh -S /tmp/ssh-device -O exit user@10.2.0.107 > /dev/null
done