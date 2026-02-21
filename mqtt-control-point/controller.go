package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"mobile.dani.df/logging"
	"mobile.dani.df/mqtt"
	"mobile.dani.df/utils"
)

const (
	mqttSearchTimeoutSeconds = 10
	mqttQOS                  = 0
	mqttRetained             = false
)

type MqttController struct {
	mqttConfig       mqtt.MqttConfig
	ctx              context.Context
	brokerConnection mqtt.MqttConnection
	DiscoveryTopic   string
}

var subscriptionChannels = make(map[string]chan string)

func NewMqttController(mqttBrokerHost string, discoveryTopic string) (*MqttController, error) {
	ctx := context.Background()
	ctx, log := logging.Init(ctx, slog.LevelDebug)

	mqttConfig := mqtt.MqttConfig{
		MqttBroker: mqttBrokerHost,
	}

	conn, err := mqtt.CreateConnection(ctx, mqttConfig)
	if err != nil {
		log.Error("[mqtt-controller] Error while creating a connection to the broker: " + err.Error())
		return nil, err
	}

	return &MqttController{
		mqttConfig:       mqttConfig,
		ctx:              ctx,
		brokerConnection: conn,
		DiscoveryTopic:   discoveryTopic,
	}, nil
}

func (controller MqttController) Search() []mqtt.Device {
	log := controller.ctx.Value("logger").(logging.Logger)

	result := []mqtt.Device{}

	wait := make(chan bool)

	handler := func(message mqtt.MqttMessage) {
		log.Debug("[mqtt-controller] {" + message.Topic + "}: <" + message.Payload + ">")
		mqttDevice := mqtt.ParseDiscoveryMessage(message)

		subscriptionChannels[mqttDevice.StateTopic] = make(chan string, 128)

		err := controller.brokerConnection.Subscribe(controller.ctx, mqttDevice.StateTopic, mqttQOS, listenSubscriptionHandler)
		if err != nil {
			log.Error("[mqtt-controller] Error while subscribing to state topic: " + mqttDevice.StateTopic)
			return
		}

		mqttDevice.SetStateFunc = func(value string) error {
			controller.brokerConnection.SendMessage(mqttDevice.CommandTopic, mqttQOS, mqttRetained, value)
			return nil
		}

		mqttDevice.GetStateFunc = func() (string, error) {
			state := <-subscriptionChannels[mqttDevice.StateTopic]
			return state, nil
		}

		result = append(result, mqttDevice)
	}
	controller.brokerConnection.Subscribe(controller.ctx, controller.DiscoveryTopic, 0, handler)

	utils.AlertAfter(mqttSearchTimeoutSeconds*time.Second, wait)

	<-wait

	return result
}

func listenSubscriptionHandler(message mqtt.MqttMessage) {
	subscriptionChannels[message.Topic] <- message.Payload
}

func (controller MqttController) PublishSwitchDevice(device *mqtt.Device) error {
	log := controller.ctx.Value("logger").(logging.Logger)

	discoveryTopic := controller.DiscoveryTopic
	if discoveryTopic[len(discoveryTopic)-1] == '#' {
		discoveryTopic = discoveryTopic[:len(discoveryTopic)-1]
	}
	if discoveryTopic[len(discoveryTopic)-1] != '/' {
		discoveryTopic = fmt.Sprintf("%s/", discoveryTopic)
	}

	id, err := mqtt.GenerateID()
	if err != nil {
		log.Error("[mqtt-config] Error while generating ID: " + err.Error())
		return err
	}

	discoveryTopic = fmt.Sprintf("%sswitch/%s/config", discoveryTopic, id)
	message, err := json.Marshal(device)
	if err != nil {
		log.Error("[mqtt-config] Error while marshaling device: " + err.Error())
		return err
	}

	subscriptionChannels[device.CommandTopic] = make(chan string)

	handler := func(message mqtt.MqttMessage) {
		subscriptionChannels[device.CommandTopic] <- message.Payload
	}

	controller.brokerConnection.Subscribe(controller.ctx, device.CommandTopic, byte(device.Qos), handler)

	device.GetRequiredState = func() string {
		return <-subscriptionChannels[device.CommandTopic]
	}

	device.AdvertiseStateFunc = func(value string) error {
		controller.brokerConnection.SendMessage(device.StateTopic, byte(device.Qos), false, value)
		return nil
	}

	controller.brokerConnection.SendMessage(discoveryTopic, byte(device.Qos), mqttRetained, string(message))

	return nil
}
