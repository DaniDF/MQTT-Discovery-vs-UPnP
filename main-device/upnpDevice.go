package main

import (
	"context"
	"net"
	"strconv"

	"mobile.dani.df/logging"
	upnp "mobile.dani.df/upnp-device"
)

const (
	upnpPort              = 8080
	devicePresentationUrl = "/Device.xml"
)

var rootDevice = upnp.RootDevice{
	SpecVersion: upnp.SpecVersion{
		Major: "0",
		Minor: "1",
	},
	Devices: []upnp.Device{
		{
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
			PresentationURL:  "http://" + GetLocalIP() + ":" + strconv.Itoa(upnpPort) + devicePresentationUrl,
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
					ServiceType: "urn:schemas-upnp-org:service:SwitchPower:1",
					ServiceId:   "urn:upnp-org:serviceId:SwitchPower",
					SCPDURL:     "/SwitchPower",
					EventSubURL: "/SwitchPower/event",
					ControlURL:  "/SwitchPower/control",
				},
				{
					ServiceType: "urn:schemas-upnp-org:service:TemperatureSensor:1",
					ServiceId:   "urn:upnp-org:serviceId:TemperatureSensor",
					SCPDURL:     "/TemperatureSensor",
					EventSubURL: "/TemperatureSensor/event",
					ControlURL:  "/TemperatureSensor/control",
				},
			},
		},
	},
}

type UDPPacket struct {
	source   net.UDPAddr
	receiver net.UDPAddr
	message  string
}

func (m UDPPacket) String() string {
	return m.source.String() + " says " + m.message
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ctx = logging.Init(ctx)

	HttpServer(ctx)
	Ssdp(ctx)

	cancel() //TODO Find a solution it is unused
}
