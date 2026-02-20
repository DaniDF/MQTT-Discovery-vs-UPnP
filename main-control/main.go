package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	device "mobile.dani.df/device-service"
	"mobile.dani.df/logging"
	"mobile.dani.df/upnp-control-point"
)

func main() {
	ctx := context.Background()
	ctx, log := logging.Init(ctx, slog.LevelDebug)

	/*
		//MQTT
		mqttConn, err := mqtt.NewMqttController("tcp://mqtt.df:1883")
		if err != nil {
			log.Error("Error while connecting to mqtt broker:" + err.Error())
			return
		}
		mqttDevices := mqttConn.Search()
		for i, dev := range mqttDevices {
			fmt.Println(strconv.Itoa(i) + ") " + dev.String() + "\n\n")
			dev.ControlFunc(device.Argument{
				Name:  "Var1",
				Value: "69",
			})
		}*/

	//UPNP
	upnpConn := upnp.NewUpnpController()
	startSearchTime := time.Now()

	//upnpDevices := upnpConn.Search()
	upnpDevices := upnpConn.SearchBySt("urn:schemas-upnp-org:device:BinaryLight:1")

	elapsedTime := time.Since(startSearchTime)
	log.Info("Found " + strconv.Itoa(len(upnpDevices)) + " devices")
	log.Info("SEARCH Elapsed time: " + elapsedTime.String())

	for i, dev := range upnpDevices {
		go func() {
			startControlTime := time.Now()
			response := dev.ControlFunc(device.Argument{
				Name:  "Var1",
				Value: "69",
			})
			elapsedTime := time.Since(startControlTime)
			log.Info(dev.Name() + " -> Control time: " + elapsedTime.String() + " resp: " + strconv.Itoa(response.ErrorCode) + " - " + response.ErrorMessage + " - " + response.Value)
		}()

		fmt.Println(strconv.Itoa(i) + ") " + dev.String() + "\n\n")

	}

	/*
		// Start - SSDP
		//rootDevices, err := upnp.Search(ctx, "urn:schemas-upnp-org:device:BinaryLight:1")
		startSearchTime := time.Now()
		rootDevices, err := upnp.Search("ssdp:all")
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

		testRootDevice := rootDevices["uuid:3bdfa216-a7c2-42ed-b50d-84e523d9761a"]
		testService := testRootDevice.Device.Services[0]

		type TurnArgs struct {
			StateValue string `xml:"StateValue"`
		}
		type TurnReply struct {
			ActualValue string `xml:"ActualValue"`
		}
		// End - SSDP

		// Start - GENA
		cancelP, err := upnp.Subscribe(ctx, testRootDevice, testService, func(s string) {
			log.Info(("Event received: " + s))
		})
		if err != nil {
			log.Error("Error subscribing to " + testService.ServiceId + ", " + err.Error())
			return
		}
		cancel := *cancelP
		// End - GENA

		time.Sleep(2 * time.Second)

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
	*/
	<-time.After(2 * time.Second)
	//cancel()
}
