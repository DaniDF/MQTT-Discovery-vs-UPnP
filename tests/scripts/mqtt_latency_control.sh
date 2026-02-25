#!/bin/bash

## Test latency

for mqtt_devices in {1..100} ;
do
    for qos in {0..2} ;
    do
        ssh -f -MS /tmp/ssh-device user@10.2.0.107 "tests/bin/device -u 0 --mqtt-broker 10.2.0.108:1883 -m $mqtt_devices --qos "$qos" > tests/logs/device_test-latency_m-"$mqtt_devices"_q-"$qos"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null"

        for mqtt_controls in {1..100}
        do
            echo "Running mqtt_devices: $mqtt_devices, mqtt_controls $mqtt_controls, qos: $qos"
            
            for count in {1..50} ;
            do
                ../bin/./control -u 0 --mqtt-broker 10.2.0.108:1883 -m "$mqtt_controls" --qos "$qos" > ../logs/control_test-latency_d-"$mqtt_devices"_m-"$mqtt_controls"_q-"$qos"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null
            done
        done

        ssh -S /tmp/ssh-device -O exit user@10.2.0.107
    done
done