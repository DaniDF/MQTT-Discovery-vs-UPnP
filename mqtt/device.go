package mqtt

import (
	"strconv"
	"strings"

	device "mobile.dani.df/device-service"
	"mobile.dani.df/utils"
)

type Device struct {
	CommandTopic string `json:"command_topic"`
	StateTopic   string `json:"state_topic"`
	Id           string `json:"-"` //TODO verify: "-" or simply - ?
	Qos          int    `json:"qos"`

	SwitchRootDevice *SwitchRootDevice
	SensorRootDevice *SensorRootDevice

	SetStateFunc func(value string) error
	GetStateFunc func() (string, error)
}

func (dev Device) ControlFunc(arguments ...device.Argument) device.Response {
	if len(arguments) == 0 {
		return device.Response{
			ErrorCode:    101,
			ErrorMessage: "Invalid arguments",
		}
	}

	err := dev.SetStateFunc(arguments[0].Value)
	if err != nil {
		return device.Response{
			ErrorCode:    102,
			ErrorMessage: err.Error(),
		}
	}

	return dev.StateFunc()
}

func (dev Device) StateFunc() device.Response {
	result, err := dev.GetStateFunc()
	if err != nil {
		return device.Response{
			ErrorCode:    100,
			ErrorMessage: err.Error(),
		}
	}

	return device.Response{
		ErrorCode: 0,
		Value:     result,
	}
}

func (dev Device) Name() string {
	return dev.Name()
}

type SwitchRootDevice struct {
	Device
	Availability    Availability `json:"availability"`
	CommandTemplate string       `json:"command_template"`

	DefaultEntityId        string         `json:"default_entity_id"`
	EmbeddedDevice         EmbeddedDevice `json:"device"`
	EnabledByDefault       bool           `json:"enabled_by_default"`
	Encoding               string         `json:"encoding"`
	EntityCategory         string         `json:"entity_category"`
	EntityPicture          string         `json:"entity_picture"`
	Icon                   string         `json:"icon"`
	JsonAttributesTemplate string         `json:"json_attributes_template"`
	JsonAttributesTopic    string         `json:"json_attributes_topic"`
	Name                   string         `json:"name"`
	Optimistic             bool           `json:"optimistic"`
	PayloadAvailable       string         `json:"payload_available"`
	PayloadNotAvailable    string         `json:"payload_not_available"`
	Payloadoff             string         `json:"payload_off"`
	PayloadOn              string         `json:"payload_on"`
	Platform               string         `json:"platform"`
	Qos                    int            `json:"qos"`
	Retain                 bool           `json:"retain"`
	StateOff               string         `json:"state_off"`
	StateOn                string         `json:"state_on"`

	UniqueId      string `json:"unique_id"`
	ValueTemplate string `json:"value_template"`
}

type SensorRootDevice struct{}

type Availability struct {
	PayloadAvailable    string `json:"payload_available"`
	PayloadNotAvailable string `json:"payload_not_available"`
	Topic               string `json:"topic"`
	ValueTemplate       string `json:"value_template"`
}

type EmbeddedDevice struct {
	ConfigurationUrl string       `json:"configuration_url"`
	Connections      []connection `json:"connections"`
	HwVersion        string       `json:"hw_version"`
	Identifiers      []string     `json:"identifiers"`
	Manufacturer     string       `json:"manufacturer"`
	Model            string       `json:"model"`
	ModelId          string       `json:"model_id"`
	Name             string       `json:"name"`
	SerialNumber     string       `json:"serial_number"`
	SuggestedArea    string       `json:"suggested_area"`
	SwVersion        string       `json:"sw_version"`
	ViaDevice        string       `json:"via_device"`
}

type connection struct {
	ConnectionType       string `json:"connection_type"`
	ConnectionIdentifier string `json:"connection_identifier"`
}

func (device Device) String() string {
	result := ""

	if device.SwitchRootDevice != nil {
		result = device.SwitchRootDevice.String()

	} else if device.SensorRootDevice != nil {
		result = device.SensorRootDevice.String()
	}

	return result
}

func (rootDevice SwitchRootDevice) String() string {
	var result strings.Builder

	result.WriteString("availability: " + rootDevice.Availability.String() + "\n")
	result.WriteString("command_template: " + rootDevice.CommandTemplate + "\n")
	result.WriteString("command_topic: " + rootDevice.CommandTopic + "\n")
	result.WriteString("default_entity_id: " + rootDevice.DefaultEntityId + "\n")
	result.WriteString("device: " + rootDevice.EmbeddedDevice.String() + "\n")
	result.WriteString("enabled_by_default: " + strconv.FormatBool(rootDevice.EnabledByDefault) + "\n")
	result.WriteString("encoding: " + rootDevice.Encoding + "\n")
	result.WriteString("entity_category: " + rootDevice.EntityCategory + "\n")
	result.WriteString("entity_picture: " + rootDevice.EntityPicture + "\n")
	result.WriteString("icon: " + rootDevice.Icon + "\n")
	result.WriteString("json_attributes_template: " + rootDevice.JsonAttributesTemplate + "\n")
	result.WriteString("json_attributes_topic: " + rootDevice.JsonAttributesTemplate + "\n")
	result.WriteString("name: " + rootDevice.Name + "\n")
	result.WriteString("optimistic: " + strconv.FormatBool(rootDevice.Optimistic) + "\n")
	result.WriteString("payload_available: " + rootDevice.PayloadAvailable + "\n")
	result.WriteString("payload_not_available: " + rootDevice.PayloadNotAvailable + "\n")
	result.WriteString("payload_off: " + rootDevice.Payloadoff + "\n")
	result.WriteString("payload_on: " + rootDevice.PayloadOn + "\n")
	result.WriteString("platform: " + rootDevice.Platform + "\n")
	result.WriteString("qos: " + strconv.Itoa(rootDevice.Qos) + "\n")
	result.WriteString("retain: " + strconv.FormatBool(rootDevice.Retain) + "\n")
	result.WriteString("state_off: " + rootDevice.StateOff + "\n")
	result.WriteString("state_on: " + rootDevice.StateOn + "\n")
	result.WriteString("state_topic: " + rootDevice.StateTopic + "\n")
	result.WriteString("unique_id: " + rootDevice.UniqueId + "\n")
	result.WriteString("value_template: " + rootDevice.ValueTemplate + "\n")

	return result.String()
}

func (sensor SensorRootDevice) String() string {
	return ""
}

func (availability Availability) String() string {
	var result strings.Builder

	result.WriteString("payload_available: " + availability.PayloadAvailable + "\n")
	result.WriteString("payload_not_available: " + availability.PayloadNotAvailable + "\n")
	result.WriteString("topic: " + availability.Topic + "\n")
	result.WriteString("value_template: " + availability.ValueTemplate + "\n")

	return result.String()
}

func (device EmbeddedDevice) String() string {
	var result strings.Builder

	result.WriteString("configuration_url: " + device.ConfigurationUrl + "\n")
	result.WriteString("connections: " + utils.StringToCSV(stringConnections(device.Connections)) + "\n")
	result.WriteString("hw_version: " + device.HwVersion + "\n")
	result.WriteString("identifiers: " + utils.StringToCSV(device.Identifiers) + "\n")
	result.WriteString("manufacturer: " + device.Manufacturer + "\n")
	result.WriteString("model: " + device.Model + "\n")
	result.WriteString("model_id: " + device.ModelId + "\n")
	result.WriteString("name: " + device.Name + "\n")
	result.WriteString("serial_number: " + device.SerialNumber + "\n")
	result.WriteString("sw_version: " + device.SwVersion + "\n")
	result.WriteString("via_device: " + device.ViaDevice + "\n")

	return result.String()
}

func stringConnections(connections []connection) []string {
	result := []string{}

	for _, connection := range connections {
		result = append(result, connection.ConnectionIdentifier+":"+connection.ConnectionType)
	}

	return result
}
