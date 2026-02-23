package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/alexflint/go-arg"

	"mobile.dani.df/logging"
	mqtt "mobile.dani.df/mqtt-control-point"
	"mobile.dani.df/upnp-control-point"
)

const (
	mqttBrokerHost     = "mqtt.df:1883"
	mqttDiscoveryTopic = "test4/#"
	mqttAliveTopic     = "test/alive"
	mqttQos            = 0
)

type Args struct {
	EnableUpnpTest bool `arg:"-u,--upnp-test" default:"false" help:"Enable UPnP test"`
	EnableMqttTest bool `arg:"-m,--mqtt-test" default:"false" help:"Enable MQTT test"`

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
	ctx, log := logging.Init(ctx, debugLevel)

	if args.EnableMqttTest {
		mqttController, err := mqtt.NewMqttController(ctx, mqttBrokerHost, mqttDiscoveryTopic, mqttAliveTopic, mqttQos)
		if err != nil {
			log.Error("[main-control] Error while connecting to mqtt broker: " + err.Error())
		}

		startSearchTime := time.Now()
		mqttDevices := mqttController.Search()
		stopSearchTime := time.Since(startSearchTime)
		log.Info("[main-control] Mqtt search found " + strconv.Itoa(len(mqttDevices)) + " devices in: " + stopSearchTime.String())
		for _, dev := range mqttDevices {
			go func() {
				startTime := time.Now()
				dev.SetStateFunc("1")
				dev.GetStateFunc()
				elapsedTime := time.Since(startTime)
				log.Info("[main-control] Device " + dev.Id + " elapsed set: " + elapsedTime.String())
			}()
		}
	}

	if args.EnableMqttTest {
		time.Sleep(2 * time.Minute)
	}

	if args.EnableUpnpTest {
		// Start - SSDP
		//rootDevices, err := upnp.Search(ctx, "urn:schemas-upnp-org:device:BinaryLight:1")
		startSearchTime := time.Now()
		rootDevices, err := upnp.Search(ctx, "urn:schemas-upnp-org:device:BinaryLight:1")
		if err != nil {
			log.Error("Error fetching rootDevices: " + err.Error())
			return
		}
		elapsedTime := time.Since(startSearchTime)
		log.Info("Found " + strconv.Itoa(len(rootDevices)) + " devices")
		log.Info("SEARCH Elapsed time: " + elapsedTime.String())

		i := 0
		for _, rootDevice := range rootDevices {
			fmt.Println(strconv.Itoa(i) + ") " + upnp.StringRootDevice(rootDevice))
			i++
		}

		keys := make([]string, 0, len(rootDevices))
		for k := range rootDevices {
			keys = append(keys, k)
		}

		testRootDevice := rootDevices[keys[len(keys)-1]]
		testService := testRootDevice.Device.Services[0]

		type TurnArgs struct {
			StateValue string `xml:"StateValue"`
		}
		type TurnReply struct {
			ActualValue string `xml:"ActualValue"`
		}
		// End - SSDP

		/*
			// Start - GENA
			cancelP, err := upnp.Subscribe(ctx, testRootDevice, testService)
			if err != nil {
				log.Error("Error subscribing to " + testService.ServiceId + ", " + err.Error())
				return
			}
			cancel := *cancelP
			// End - GENA

			time.Sleep(2 * time.Second)
		*/

		// Start - SOAP
		startRPCTime := time.Now()
		soap := testService.NewSOAPClient()
		soapArgs := TurnArgs{
			StateValue: "1",
		}
		reply := TurnReply{}
		err = soap.PerformActionCtx(ctx, testService.ServiceType, "Turn", &soapArgs, &reply)
		if err != nil {
			log.Error("Error RPC: " + err.Error())
		}
		log.Info("RPC returned: " + reply.ActualValue)

		elapsedTime = time.Since(startRPCTime)
		log.Info("SOAP Elapsed time: " + elapsedTime.String())
		// End - SOAP

		/*
			<-time.After(20 * time.Second)
			cancel()
		*/
	}
}
