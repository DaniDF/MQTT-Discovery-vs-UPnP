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

func ParseDiscoveryMessage(message MqttMessage, discoveryPrefix string) (Device, error) {
	result, err := parseDevice(message.Payload)
	if err != nil {
		return Device{}, err
	}

	prefix := strings.TrimSuffix(discoveryPrefix, "#")
	prefix = strings.TrimSuffix(prefix, "/") + "/"
	result.Id = strings.Split(strings.TrimPrefix(message.Topic, prefix), "/")[1]

	return result, nil
}

func parseDevice(message string) (Device, error) {
	result := Device{}
	err := json.Unmarshal([]byte(message), &result)
	return result, err
}
