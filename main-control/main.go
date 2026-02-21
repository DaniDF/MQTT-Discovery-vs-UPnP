package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"mobile.dani.df/logging"

	"mobile.dani.df/upnp-control-point"
)

func main() {
	ctx := context.Background()
	ctx, log := logging.Init(ctx, slog.LevelDebug)

	/*mqttConfig := mqtt.MqttConfig{
		MqttBroker: "tcp://mqtt.df:1883",
	}

	conn, err := mqtt.CreateConnection(ctx, mqttConfig)

	if err != nil {
		panic("Can not continue without a connection")
	}

	conn.Subscribe("homeassistant/#", 0, printMessage)

	for i := range 10 {
		conn.SendMessage("test/publish", 0, false, "Testttt "+strconv.Itoa(i))
	}

	time.Sleep(10 * time.Second)

	conn.Unsubscribe("homeassistant/#")*/

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
	args := TurnArgs{
		StateValue: "1",
	}
	reply := TurnReply{}
	err = soap.PerformActionCtx(ctx, testService.ServiceType, "Turn", &args, &reply)
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
