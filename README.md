# MQTT-Discovery vs UPnP

This project aims to compare two discovery protocols: [MQTT-Discovery](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery)(MQTT-D) and [UPnP](https://openconnectivity.org/developer/specifications/upnp-resources/upnp/)

## 💠FYI

Here a quick list of files that may interest you:

* [report](https://github.com/DaniDF/MQTT-Discovery-vs-UPnP/blob/main/docs/MQTTDiscovery_vs_UPnP.pdf): the report of the experiments in a IEEE format
* [analysis](https://github.com/DaniDF/MQTT-Discovery-vs-UPnP/blob/main/tests/analysis/data_analysis.ipynb): the analysis file (python notebook) to reproduce the evaluations made in the report



## 💠 Overview

This repo refers to an academic study (related to the course "Mobile Systems M" at University of Bologna) that compares latency and robustness of MQTT-D and UPnP.

The chosen protocols are quite different:

* MQTT-D: based on MQTT, pub-sub message exchange protocol
* UPnP: peer-to-peer messaging system that via General Event Notification Architecture (GENA) is able of event delivery.



## 💠 Usage

* Run the device:

  ```sh
  go run main-device/main.go [-u number_of_upnp_devices] [-m number_of_mqtt_devices --mqtt-broker mqtt_broker_ip:mqttbroker_port --qos qos_level]
  ```

* Run the control:

  ```sh
  go run main-control/main.go [-u number_of_upnp_devices] [-m number_of_mqtt_devices --mqtt-broker mqtt_broker_ip:mqttbroker_port --qos qos_level]
  ```



## 💠 Run experiments

The experiments to run need a password less ssh connection between nodes. All the tests can be run in the same machine but it is suggested to launch devices and controls on different nodes.

```sh
# For UPnP test
cd tests/scripts
sh -c upnp_latency_control.sh
# For MQTT test
cd tests/scripts
sh -c mqtt_latency_control.sh
```

This test run simultaneously latency and robustness. These scripts generate log files (a lot) with all the metrics of all test runs.



## 💠 Test analysis

To analyse all the produced log files see the python notebook in [data_analysis.ipynb](https://github.com/DaniDF/MQTT-Discovery-vs-UPnP/blob/main/tests/analysis/data_analysis.ipynb)



## 💠 Report

See the [documentation](https://github.com/DaniDF/MQTT-Discovery-vs-UPnP/blob/main/docs/MQTTDiscovery_vs_UPnP.pdf) to learn how the experiments were conducted and the obtained results.



## 📄 License

Distributed under the AGPLv3 License. See [licence](https://github.com/DaniDF/MQTT-Discovery-vs-UPnP/blob/main/LICENCE.md) for more information.