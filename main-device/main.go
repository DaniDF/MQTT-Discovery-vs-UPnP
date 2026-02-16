package main

import (
	"context"
	"strconv"
	"strings"

	"mobile.dani.df/logging"
	upnp "mobile.dani.df/upnp"
)

const (
	upnpPort              = 8080
	devicePresentationUrl = "/device.xml"
)

var stateVariable = upnp.StateVariable{
	SendEvents:        true,
	Multicast:         false,
	Name:              "state",
	DataType:          "int",
	DefaultValue:      "0",
	AllowedValueRange: nil,
	AllowedValueList:  nil,
}

var spcd = upnp.Spcd{
	SpecVersion: upnp.SpecVersion{
		Major: "1",
		Minor: "1",
	},
	ServiceStateTable: []*upnp.StateVariable{&stateVariable},
}

var rootDevice = upnp.RootDevice{
	SpecVersion: upnp.SpecVersion{
		Major: "0",
		Minor: "1",
	},
	Device: upnp.Device{
		DeviceType:       "urn:schemas-upnp-org:device:BinaryLight:1",
		UDN:              "uuid:55076f6e-6b79-4d65-6401-00d0b811d10b", //TODO to be generate random (see specification 1.1.4)
		FriendlyName:     "SmartLight",
		Manufacturer:     "DF Corp.",
		ManufacturerURL:  "http://superlight.df",
		ModelName:        "SmartLight pro plus",
		ModelURL:         "http://superlight.df/smartlight-pro-plus",
		ModelDescription: "The best smart light",
		ModelNumber:      "422",
		SerialNumber:     "123-456-789-0",
		UPC:              "12345678900987654321",
		PresentationURL:  "http://" + upnp.GetLocalIP() + ":" + strconv.Itoa(upnpPort) + devicePresentationUrl,
		IconList: []upnp.Icon{
			{
				Mimetype: "image/jpeg",
				Height:   "48",
				Width:    "48",
				Depth:    "24",
				Url:      "/images/icon-48x48.jpg",
			},
			{
				Mimetype: "image/jpeg",
				Height:   "120",
				Width:    "120",
				Depth:    "24",
				Url:      "/images/icon-120x120.jpg",
			},
		},
		ServiceList: []upnp.Service{
			{
				ServiceType:    "urn:schemas-upnp-org:service:SwitchPower:1",
				ServiceId:      "urn:upnp-org:serviceId:SwitchPower",
				SCPDURL:        "/SwitchPower",
				EventSubURL:    "/SwitchPower/event",
				ControlURL:     "/SwitchPower/control",
				ControlHandler: mockControlHandler,
				SCPD:           spcd,
			},
			{
				ServiceType:    "urn:schemas-upnp-org:service:TemperatureSensor:1",
				ServiceId:      "urn:upnp-org:serviceId:TemperatureSensor",
				SCPDURL:        "/TemperatureSensor",
				EventSubURL:    "/TemperatureSensor/event",
				ControlURL:     "/TemperatureSensor/control",
				ControlHandler: mockControlHandler,
			},
		},
		EmbeddedDevices: []upnp.Device{},
	},
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ctx = logging.Init(ctx)

	log := ctx.Value("logger").(logging.Logger)

	err := spcd.AddAction(upnp.Action{
		Name: "Turn on",
		ArgumentList: []upnp.Argument{
			{
				Name:                 "StateValue",
				Direction:            upnp.In,
				RelatedStateVariable: &stateVariable,
			},
		},
	})
	if err != nil {
		log.Error("Error adding action " + err.Error())
	}

	rootDevice.Device.ServiceList[0].SCPD = spcd

	upnp.HttpServer(ctx, rootDevice, devicePresentationUrl)
	upnp.SsdpDevice(ctx, rootDevice)

	cancel() //TODO Find a solution it is unused
}

func mockControlHandler() string {
	var result strings.Builder

	/*
		<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
			<s:Body>
				<u:actionNameResponse xmlns:u="urn:schemas-upnp-org:service:serviceType:v">
					<argumentName>out arg value</argumentName>
					<!-- other out args and their values go here, if any -->
				</u:actionNameResponse>
			</s:Body>
		</s:Envelope>
	*/

	result.WriteString("<s:Envelope xmlns:s=\"http://schemas.xmlsoap.org/soap/envelope/\" s:encodingStyle=\"http://schemas.xmlsoap.org/soap/encoding/\">\n")
	result.WriteString("<s:Body>\n")
	result.WriteString("<u:actionNameResponse xmlns:u=\"urn:schemas-upnp-org:service:serviceType:v\">\n")
	result.WriteString("<argumentName>out arg value</argumentName>\n")
	result.WriteString("<!-- other out args and their values go here, if any -->\n")
	result.WriteString("</u:actionNameResponse>\n")
	result.WriteString("</s:Body>\n")
	result.WriteString("</s:Envelope>\n")

	return result.String()
}
