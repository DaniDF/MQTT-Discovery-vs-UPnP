package main

import (
	"context"
	"strconv"

	"mobile.dani.df/logging"

	"mobile.dani.df/upnp-control-point"
	//"mobile.dani.df/upnp"
)

func main() {
	ctx := context.Background()
	ctx, log := logging.Init(ctx)

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

	rootDevice, err := upnp.Search(ctx, "urn:schemas-upnp-org:device:BinaryLight:1")
	if err != nil {
		log.Error("Error fetching rootDevices: " + err.Error())
		return
	}
	log.Info("Found " + strconv.Itoa(len(rootDevice)) + " devices")

	testRootDevice := rootDevice[len(rootDevice)-1]
	testService := testRootDevice.Device.Services[0]

	type TurnArgs struct {
		StateValue string `xml:"StateValue"`
	}
	type TurnReply struct {
	}

	soap := testService.NewSOAPClient()
	args := TurnArgs{
		StateValue: "1",
	}
	reply := TurnReply{}
	err = soap.PerformActionCtx(ctx, testService.ServiceType, "Turn", &args, &reply)
	if err != nil {
		log.Error("Error RPC: " + err.Error())
	}
}
