package main

import (
	"context"
	"log/slog"
	"strconv"

	"mobile.dani.df/device-service"
	"mobile.dani.df/logging"
	upnp "mobile.dani.df/upnp"
	"mobile.dani.df/utils"
)

const (
	upnpPort              = 8080
	devicePresentationUrl = "/device.xml"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ctx, _ = logging.Init(ctx, slog.LevelDebug)

	rootDevice, err := CreateUpnpRootDevice(ctx)
	if err != nil {
		return
	}

	upnp.GenaSubscriptionDaemon(ctx)
	upnp.HttpServer(ctx, rootDevice, devicePresentationUrl)
	upnp.SsdpDevice(ctx, rootDevice)

	cancel() //TODO Find a solution it is unused
}

func CreateUpnpRootDevice(ctx context.Context) (upnp.RootDevice, error) {
	log := ctx.Value("logger").(logging.Logger)

	uuid, err := upnp.GenerateRandomUUID()
	if err != nil {
		log.Error("Error while generating the UUID for the device")
		return upnp.RootDevice{}, err
	}

	result := upnp.RootDevice{
		SpecVersion: upnp.SpecVersion{
			Major: "0",
			Minor: "1",
		},
		Device: upnp.Device{
			DeviceType:       "urn:schemas-upnp-org:device:BinaryLight:1",
			UDN:              uuid,
			FriendlyName:     "SmartLight",
			Manufacturer:     "DF Corp.",
			ManufacturerURL:  "http://superlight.df",
			ModelName:        "SmartLight pro plus",
			ModelURL:         "http://superlight.df/smartlight-pro-plus",
			ModelDescription: "The best smart light",
			ModelNumber:      "422",
			SerialNumber:     "123-456-789-0",
			UPC:              "12345678900987654321",
			PresentationURL:  "http://" + utils.GetLocalIP() + ":" + strconv.Itoa(upnpPort) + devicePresentationUrl,
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
			EmbeddedDevices: []upnp.Device{},
		},
	}

	stateValueVariable := upnp.StateVariable{
		SendEvents:        true,
		Multicast:         false,
		Name:              "state",
		DataType:          "string",
		DefaultValue:      "0",
		AllowedValueRange: nil,
		AllowedValueList:  nil,
	}

	actualValueVariable := upnp.StateVariable{
		SendEvents:        true,
		Multicast:         false,
		Name:              "actualState",
		DataType:          "string",
		DefaultValue:      "0",
		AllowedValueRange: nil,
		AllowedValueList:  nil,
	}

	var spcd = upnp.Spcd{
		SpecVersion: upnp.SpecVersion{
			Major: "1",
			Minor: "1",
		},
		ServiceStateTable: []*upnp.StateVariable{&stateValueVariable, &actualValueVariable},
	}

	err = spcd.AddAction(upnp.FormalAction{
		Name: "Turn",
		ArgumentList: []upnp.FormalArgument{
			{
				Name:                 "StateValue",
				Direction:            upnp.In,
				RelatedStateVariable: &stateValueVariable,
			},
			{
				Name:                 "ActualValue",
				Direction:            upnp.Out,
				RelatedStateVariable: &actualValueVariable,
			},
		},
	})
	if err != nil {
		log.Error("Error adding action " + err.Error())
	}
	err = spcd.AddAction(upnp.FormalAction{
		Name: "Read",
		ArgumentList: []upnp.FormalArgument{
			{
				Name:                 "ActualValue",
				Direction:            upnp.Out,
				RelatedStateVariable: &actualValueVariable,
			},
		},
	})
	if err != nil {
		log.Error("Error adding action " + err.Error())
	}

	result.Device.ServiceList[0].SCPD = spcd
	result.Device.ServiceList[0].Handler = func(arguments ...device.Argument) device.Response { //TODO I should have a reference on which action is invoked
		log.Debug("[service] Execute service: urn:upnp-org:serviceId:SwitchPower action: Turn value: " + arguments[0].Value)

		switch arguments[0].Value {
		case "0", "1":
			upnp.NotifySubscribers(result.Device.ServiceList[0], []device.Argument{arguments[0]})
			return device.Response{
				Value: arguments[0].Value,
			}
		default:
			return device.Response{
				ErrorCode:    101,
				ErrorMessage: "Test application error",
			}
		}
	}

	return result, nil
}
