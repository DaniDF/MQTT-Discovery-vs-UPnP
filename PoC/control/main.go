package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/alexflint/go-arg"

	"github.com/DaniDF/MQTT-Discovery-vs-UPnP/logging"
	mqtt "github.com/DaniDF/MQTT-Discovery-vs-UPnP/mqtt-control-point"
	"github.com/DaniDF/MQTT-Discovery-vs-UPnP/upnp-control-point"
)

const (
	mqttDiscoveryTopic = "test/discovery/#"
	mqttAliveTopic     = "test/alive"

	mqttControlsTimeout = 1800 * time.Second
	mqttRPCTimeout      = 10 * time.Second

	upnpControlsTimeout = 50 * time.Second
	genaTimeout         = 10 * time.Second
)

type Args struct {
	UpnpEnabled bool `arg:"-u,--upnp" default:"false" help:"Use UPnP as discovery protocol"`
	MqttEnabled bool `arg:"-m,--mqtt" default:"false" help:"Use MQTT as discovery protocol"`

	Mx int `arg:"--mx" default:"0" help:"Set a manual value for MX"`

	MqttBroker string `arg:"--mqtt-broker" default:" "  help:"MQTT broker"`
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

	waitMqtt := make(chan bool, 1)

	if args.MqttEnabled {
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
				select {
				case <-waitMqttRPC:
				case <-time.After(mqttRPCTimeout):
					log.Warn("[main-control] Not received all RPC responses before timeout")
					waitMqtt <- true
					return
				}
			}

			waitMqtt <- true
		}()
	}

	if args.MqttEnabled {
		select {
		case <-waitMqtt:
		case <-time.After(mqttControlsTimeout):
			log.Warn("[main-control] Not mqtt controller returned before timeout")
			return
		}
	}

	mx := args.Mx
	if args.Mx <= 0 {
		mx = 2
	}

	if args.UpnpEnabled {
		testSoap(ctx, args, mx, logLevel)
		//testGena(ctx, args, mx, logLevel)
	}
}

func testSoap(ctx context.Context, args Args, mx int, logLevel slog.Level) {
	log := ctx.Value("logger").(logging.Logger)

	waitUpnp := make(chan bool, 1)
	if args.UpnpEnabled {
		go func() {
			// Start - SSDP
			startSearchTime := time.Now()
			rootDevices, err := upnp.SearchMx(ctx, "urn:schemas-upnp-org:device:BinaryLight:1", mx)
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

			for _, rootDevice := range rootDevices {
				go func() {
					testService := rootDevice.Device.Services[0]

					var startRPCTime time.Time

					waitGena := make(chan bool, 1)

					// Start - GENA
					var cancel context.CancelFunc
					cancelP, err := upnp.Subscribe(ctx, rootDevice, testService, func(event string) {
						log.Trace("[main-control] Event elapsed time: " + time.Since(startRPCTime).String())
						log.Debug("[main-control] Received event: " + event)

						cancel()
						upnp.Unsubscribe(ctx, rootDevice, testService)
						waitGena <- true

					})
					if err != nil {
						log.Error("[main-control] Error subscribing to " + testService.ServiceId + ", " + err.Error())
						waitRootDevice <- true
						return
					}
					cancel = *cancelP
					// End - GENA

					// Start - SOAP
					soap := testService.NewSOAPClient()
					soapArgs := TurnArgs{
						StateValue: "0",
					}
					reply := TurnReply{}

					startRPCTime = time.Now()
					err = soap.PerformActionCtx(ctx, testService.ServiceType, "Turn", &soapArgs, &reply)
					if err != nil {
						log.Error("[main-control] Error RPC: " + err.Error())
					}
					elapsedTime = time.Since(startRPCTime)

					log.Info("[main-control] RPC returned: " + reply.ActualValue)
					log.Trace("[main-control] RPC Elapsed time: " + elapsedTime.String())
					// End - SOAP

					select {
					case <-waitGena:
					case <-time.After(genaTimeout):
						log.Warn("[main-control] Not received gena response before timeout")
					}

					waitRootDevice <- true
				}()
			}

			for range len(rootDevices) {
				<-waitRootDevice
			}

			waitUpnp <- true
		}()
	}

	if args.UpnpEnabled {
		select {
		case <-waitUpnp:
		case <-time.After(upnpControlsTimeout + time.Duration(mx)*time.Second):
			log.Warn("[main-control] Not all the upnp controls have returned before timeout")
			return
		}
	}
}
