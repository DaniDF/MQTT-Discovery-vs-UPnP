package mqtt

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
)

func GenerateID() (string, error) {
	buffer := make([]byte, 8)
	_, err := rand.Read(buffer)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x-%x-%x-%x", buffer[:2], buffer[2:4], buffer[4:6], buffer[6:8]), nil
}

func ParseDiscoveryMessage(message MqttMessage) Device {
	/* //TODO Check if you want to maintain this division or switch to "Everythig is a Device" policy
	deviceType := strings.Split(message.Topic, "/")[1]


	result := Device{
		SwitchRootDevice: nil,
		SensorRootDevice: nil,
	}
	switch deviceType {
	case "switch":
		result.SwitchRootDevice = parseSwitchDeviceMessage(message.Payload)
	case "sensor":
		result.SensorRootDevice = parseSensorDeviceMessage(message.Payload)
	default:
		result.SwitchRootDevice = parseSwitchDeviceMessage(message.Payload)
	}

	return result
	*/

	result := parseDevice(message.Payload)
	result.Id = strings.Split(message.Topic, "/")[2]
	return result
}

func parseDevice(message string) Device {
	result := Device{}
	json.Unmarshal([]byte(message), &result)
	return result
}

func parseSwitchDeviceMessage(message string) *SwitchRootDevice {
	result := SwitchRootDevice{}
	json.Unmarshal([]byte(message), &result)
	return &result
}

func parseSensorDeviceMessage(message string) *SensorRootDevice {
	result := SensorRootDevice{}
	json.Unmarshal([]byte(message), &result)
	return &result
}
