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

for mqtt_devices in {1..9} {10..100..10} ;
do
    for qos in {0..2} ;
    do
        for mqtt_controls in {1..9} {10..100..10} ;
        do
            echo "Running mqtt_devices: $mqtt_devices, mqtt_controls $mqtt_controls, qos: $qos"

            for count in {1..30} ;
            do
                ssh -f $REMOTE_USER "tests/bin/device -m $mqtt_devices --mqtt-broker 10.2.0.108:1883 --qos "$qos" > tests/logs/device_test-latency_m-"$mqtt_devices"_q-"$qos"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null"
                
                sleep 4
                
                ../bin/./control -m "$mqtt_controls" --mqtt-broker 10.2.0.108:1883 --qos "$qos" > ../logs/control_test-latency_d-"$mqtt_devices"_m-"$mqtt_controls"_q-"$qos"_$(date +"%Y%m%d_%H%M%S_%N").log 2>&1 < /dev/null

                kill_remote_jobs
                sleep 1
            done
        done
    done
done