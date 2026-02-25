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
	mqttDiscoveryTopic = "test/discovery/#"
	mqttAliveTopic     = "test/alive"

	upnpControlsTimeout = 50 * time.Second
	genaTimeout         = 10 * time.Second
)

type Args struct {
	NumUpnpControl int `arg:"-u,--upnp-ctrl" default:"1" help:"Number of UPnP control points to deploy"`
	NumMqttControl int `arg:"-m,--mqtt-ctrl" default:"1" help:"Number of MQTT control points to deploy"`

	MqttBroker string `arg:"required,--mqtt-broker"  help:"MQTT broker"`
	MqttQos    int    `arg:"--qos" default:"0" help:"Sets the MQTT Qos"`

	DebugEnabled bool `arg:"-d,--debug" default:"false" help:"Enable debug logging"`
}

func main() {
	var args Args
	arg.MustParse(&args)

	logLevel := logging.LevelTrace
	if args.DebugEnabled {
		logLevel = slog.LevelDebug
	}

	ctx := context.Background()
	ctx, log := logging.Init(ctx, logLevel)

	waitMqttControls := make(chan bool, args.NumMqttControl)

	for range args.NumMqttControl {
		go func() {
			mqttController, err := mqtt.NewMqttController(ctx, args.MqttBroker, mqttDiscoveryTopic, mqttAliveTopic, args.MqttQos)
			if err != nil {
				log.Error("[main-control] Error while connecting to mqtt broker: " + err.Error())
			}

			startSearchTime := time.Now()
			mqttDevices := mqttController.Search()
			stopSearchTime := time.Since(startSearchTime)
			log.Trace("[main-control] Mqtt search found " + strconv.Itoa(len(mqttDevices)) + " devices in: " + stopSearchTime.String())

			waitMqttRPC := make(chan bool, len(mqttDevices))

			for _, dev := range mqttDevices {
				go func() {
					startTime := time.Now()
					dev.SetStateFunc("1")
					dev.GetStateFunc()
					elapsedTime := time.Since(startTime)
					log.Trace("[main-control] Device " + dev.Id + " elapsed set: " + elapsedTime.String())
					waitMqttRPC <- true
				}()
			}

			for range len(mqttDevices) {
				<-waitMqttRPC
			}

			waitMqttControls <- true
		}()
	}

	for range args.NumMqttControl {
		select {
		case <-waitMqttControls:
		case <-time.After(50 * time.Second):
			return
		}
	}

	waitUpnpControls := make(chan bool, args.NumUpnpControl)

	for range args.NumUpnpControl {
		go func() {
			// Start - SSDP
			startSearchTime := time.Now()
			rootDevices, err := upnp.Search(ctx, "urn:schemas-upnp-org:device:BinaryLight:1")
			if err != nil {
				log.Error("[main-control] Error fetching rootDevices: " + err.Error())
				return
			}
			elapsedTime := time.Since(startSearchTime)
			log.Info("[main-control] Found " + strconv.Itoa(len(rootDevices)) + " devices")
			log.Trace("[main-control] SEARCH Elapsed time: " + elapsedTime.String())

			if logLevel <= slog.LevelDebug {
				i := 0
				for _, rootDevice := range rootDevices {
					fmt.Println(strconv.Itoa(i) + ") " + upnp.StringRootDevice(rootDevice))
					i++
				}
			}
			// End - SSDP

			type TurnArgs struct {
				StateValue string `xml:"StateValue"`
			}
			type TurnReply struct {
				ActualValue string `xml:"ActualValue"`
			}

			waitRootDevice := make(chan bool, len(rootDevices))

			for i, rootDevice := range rootDevices {
				func() {
					testService := rootDevice.Device.Services[0]

					var startRPCTime time.Time

					//waitGena := make(chan bool, 1)

					// Start - GENA
					//var cancel context.CancelFunc
					/*
						_, err := upnp.Subscribe(ctx, rootDevice, testService, func(event string) {
							log.Trace(i + " [main-control] Event elapsed time: " + time.Since(startRPCTime).String())
							log.Debug("[main-control] Received event: " + event)

							//cancel()
							upnp.Unsubscribe(ctx, rootDevice, testService)
							waitGena <- true

						})
						if err != nil {
							log.Error("[main-control] Error subscribing to " + testService.ServiceId + ", " + err.Error())
							waitRootDevice <- true
							return
						}
					*/
					//cancel = *cancelP
					// End - GENA

					// Start - SOAP
					soap := testService.NewSOAPClient()
					soapArgs := TurnArgs{
						StateValue: "1",
					}
					reply := TurnReply{}

					startRPCTime = time.Now()
					err = soap.PerformActionCtx(ctx, testService.ServiceType, "Turn", &soapArgs, &reply)
					if err != nil {
						log.Error("[main-control] Error RPC: " + err.Error())
					}
					elapsedTime = time.Since(startRPCTime)

					log.Info("[main-control] RPC returned: " + reply.ActualValue)
					log.Trace(i + " [main-control] RPC Elapsed time: " + elapsedTime.String())
					// End - SOAP

					/*
						select {
						case <-waitGena:
						case <-time.After(genaTimeout):
							log.Warn("[main-control] Not received gena response before timeout")
						}
					*/

					waitRootDevice <- true
				}()
			}

			for range len(rootDevices) {
				<-waitRootDevice
			}

			waitUpnpControls <- true
		}()
	}

	for range args.NumUpnpControl {
		select {
		case <-waitUpnpControls:
		case <-time.After(upnpControlsTimeout):
			log.Warn("[main-control] Not all the upnp controls have returned before timeout")
			return
		}
	}
}
