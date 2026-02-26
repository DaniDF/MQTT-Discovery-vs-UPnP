package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/alexflint/go-arg"

	"mobile.dani.df/device-service"
	"mobile.dani.df/logging"
	"mobile.dani.df/mqtt"
	ctrlmqtt "mobile.dani.df/mqtt-control-point"
	upnp "mobile.dani.df/upnp"
	"mobile.dani.df/utils"
)

const (
	devicePresentationUrl = "/device.xml"
	mqttDiscoveryTopic    = "test/discovery/#"
	mqttAliveTopic        = "test/alive"
	mqttPrefix            = "mqttdevice"
)

type Args struct {
	NumUpnpDevices int `arg:"-u,--upnp-devs" default:"0" help:"Number of UPnP devices to deploy"`
	NumMqttDevices int `arg:"-m,--mqtt-devs" default:"0" help:"Number of MQTT devices to deploy"`

	MqttBroker string `arg:"--mqtt-broker" default:" "  help:"MQTT broker"`
	MqttQos    int    `arg:"--qos" default:"0" help:"Sets the MQTT Qos"`

	DebugEnabled bool `arg:"-d,--debug" default:"false" help:"Enable debug logging"`
}

func main() {
	var args Args
	arg.MustParse(&args)

	debugLevel := slog.LevelInfo
	if args.DebugEnabled {
		debugLevel = slog.LevelDebug
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ctx, log := logging.Init(ctx, debugLevel)

	if args.NumMqttDevices > 0 {
		mqttController, err := ctrlmqtt.NewMqttController(ctx, args.MqttBroker, mqttDiscoveryTopic, mqttAliveTopic, args.MqttQos)
		if err != nil {
			log.Error("[main-device] Error while creating the mqtt controller: " + err.Error())
			return
		}

		for range args.NumMqttDevices {
			mqttDevice, err := CreateMqttSwitchDevice(ctx)
			if err != nil {
				return
			}

			mqttController.PublishSwitchDevice(&mqttDevice)

			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
						mqttDevice.AdvertiseStateFunc(mqttDevice.GetRequiredState())
					}
				}
			}()
		}

		time.Sleep(time.Hour)
	}

	if args.NumUpnpDevices > 0 {
		for range args.NumUpnpDevices {
			go func() {
				gena := upnp.NewGenaListener(ctx)
				ctx = context.WithValue(ctx, "gena", gena)

				httpServer, err := upnp.NewHttpServer(ctx)
				if err != nil {
					return
				}

				rootDevice, err := CreateUpnpRootDevice(ctx, httpServer.Port)
				if err != nil {
					return
				}

				httpServer.ServeRootDevice(rootDevice, devicePresentationUrl)
				upnp.SsdpDevice(ctx, rootDevice)
			}()
		}

		time.Sleep(time.Hour)
		cancel()
	}
}

func CreateMqttSwitchDevice(ctx context.Context) (mqtt.Device, error) {
	log := ctx.Value("logger").(logging.Logger)

	id, err := mqtt.GenerateID()
	if err != nil {
		log.Error("[main-device] Error generating mqtt device id: " + err.Error())
		return mqtt.Device{}, err
	}

	commandTopic := fmt.Sprintf("%s/%s/command", mqttPrefix, id)
	stateTopic := fmt.Sprintf("%s/%s/state", mqttPrefix, id)

	setStateFunc := func(value string) error {
		return nil
	}

	return mqtt.Device{
		CommandTopic: commandTopic,
		StateTopic:   stateTopic,
		Id:           id,

		SetStateFunc: setStateFunc,
	}, nil
}

func CreateUpnpRootDevice(ctx context.Context, upnpPort int) (upnp.RootDevice, error) {
	log := ctx.Value("logger").(logging.Logger)
	gena := ctx.Value("gena").(*upnp.GenaState)

	uuid, err := upnp.GenerateRandomUUID()
	if err != nil {
		log.Error("[main-device] Error while generating the UUID for the device")
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

	var scpd = upnp.Scpd{
		SpecVersion: upnp.SpecVersion{
			Major: "1",
			Minor: "1",
		},
		ServiceStateTable: []*upnp.StateVariable{&stateValueVariable, &actualValueVariable},
	}

	err = scpd.AddAction(upnp.FormalAction{
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
		log.Error("[main-device] Error adding action " + err.Error())
		return upnp.RootDevice{}, err
	}
	/*
		err = scpd.AddAction(upnp.FormalAction{
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
			log.Error("[main-device] Error adding action " + err.Error())
			return upnp.RootDevice{}, err
		}
	*/

	result.Device.ServiceList[0].SCPD = scpd
	result.Device.ServiceList[0].Handler = func(arguments ...device.Argument) device.Response {
		log.Info("[service] Execute service: urn:upnp-org:serviceId:SwitchPower action: Turn value: " + arguments[0].Value)

		switch arguments[0].Value {
		case "0", "1":
			gena.GenaNotifySubscribers(result.Device.ServiceList[0], []device.Argument{
				{
					Name:  "actualState",
					Value: arguments[0].Value,
				},
			})
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
