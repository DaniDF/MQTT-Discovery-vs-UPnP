package main

import (
	"fmt"
	"strconv"
	"time"

	"golang.org/x/net/context"
	"mobile.dani.df/logging"
	"mobile.dani.df/mqtt"
)

func main() {
	ctx := context.Background()
	ctx = logging.Init(ctx)

	mqttConfig := mqtt.MqttConfig{
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

	conn.Unsubscribe("homeassistant/#")
}

func printMessage(message mqtt.MqttMessage) {
	fmt.Println("Received [" + message.Topic + "] " + message.Payload)
}
