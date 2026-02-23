package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mobile.dani.df/logging"
	"mobile.dani.df/mqtt"
	"mobile.dani.df/utils"
)

const (
	mqttSearchTimeoutSeconds = 10
	//mqttQOS                  = 0
	mqttRetained = false
)

type MqttController struct {
	mqttConfig       mqtt.MqttConfig
	ctx              context.Context
	brokerConnection mqtt.MqttConnection
	DiscoveryTopic   string
	AliveTopic       string
	Qos              byte
}

var subscriptionChannels = make(map[string]chan string)

func NewMqttController(ctx context.Context, mqttBrokerHost string, discoveryTopic string, aliveTopic string, qos int) (*MqttController, error) {
	log := ctx.Value("logger").(logging.Logger)

	mqttConfig := mqtt.MqttConfig{
		MqttBroker: mqttBrokerHost,
	}

	conn, err := mqtt.CreateConnection(ctx, mqttConfig)
	if err != nil {
		log.Error("[mqtt-controller] Error while creating a connection to the broker: " + err.Error())
		return nil, err
	}

	result := MqttController{
		mqttConfig:       mqttConfig,
		ctx:              ctx,
		brokerConnection: conn,
		DiscoveryTopic:   discoveryTopic,
		AliveTopic:       aliveTopic,
		Qos:              byte(qos),
	}
	result.discoveryDaemon(byte(qos))

	return &result, nil
}

func (controller MqttController) Search() []mqtt.Device {
	log := controller.ctx.Value("logger").(logging.Logger)

	result := []mqtt.Device{}

	wait := make(chan bool)

	handler := func(message mqtt.MqttMessage) {
		log.Debug("[mqtt-controller] Discovered: {" + message.Topic + "}: <" + message.Payload + ">")
		mqttDevice := mqtt.ParseDiscoveryMessage(message)

		subscriptionChannels[mqttDevice.StateTopic] = make(chan string, 128)

		err := controller.brokerConnection.Subscribe(controller.ctx, mqttDevice.StateTopic, controller.Qos, listenSubscriptionHandler)
		if err != nil {
			log.Error("[mqtt-controller] Error while subscribing to state topic: " + mqttDevice.StateTopic)
			return
		}

		mqttDevice.SetStateFunc = func(value string) error {
			controller.brokerConnection.SendMessage(mqtt.MqttMessage{
				Topic:    mqttDevice.CommandTopic,
				Qos:      controller.Qos,
				Retained: mqttRetained,
				Payload:  value,
			})
			return nil
		}

		mqttDevice.GetStateFunc = func() (string, error) {
			state := <-subscriptionChannels[mqttDevice.StateTopic]
			return state, nil
		}

		result = append(result, mqttDevice)
	}
	controller.brokerConnection.SendMessage(mqtt.MqttMessage{
		Topic:    controller.AliveTopic,
		Qos:      controller.Qos,
		Retained: false,
		Payload:  "alive",
	})

	controller.brokerConnection.Subscribe(controller.ctx, controller.DiscoveryTopic, 0, handler)

	utils.AlertAfter(mqttSearchTimeoutSeconds*time.Second, wait)

	<-wait

	return result
}

func listenSubscriptionHandler(message mqtt.MqttMessage) {
	subscriptionChannels[message.Topic] <- message.Payload
}

var publishQueue = []mqtt.MqttMessage{}

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
		controller.brokerConnection.SendMessage(mqtt.MqttMessage{
			Topic:    device.StateTopic,
			Qos:      byte(device.Qos),
			Retained: mqttRetained,
			Payload:  value,
		})
		return nil
	}

	publishQueue = append(publishQueue, mqtt.MqttMessage{
		Topic:    discoveryTopic,
		Qos:      byte(device.Qos),
		Retained: mqttRetained,
		Payload:  string(message),
	})
	//controller.brokerConnection.SendMessage(discoveryTopic, byte(device.Qos), mqttRetained, string(message))

	return nil
}

func (controller MqttController) discoveryDaemon(qos byte) {
	log := controller.ctx.Value("logger").(logging.Logger)

	discoverySearch := make(chan bool)
	controller.brokerConnection.Subscribe(controller.ctx, controller.AliveTopic, qos, func(message mqtt.MqttMessage) {
		log.Info("[mqtt-controller] Received alive message")
		discoverySearch <- true
	})

	go func() {
		for {
			select {
			case <-controller.ctx.Done():
				return
			case <-discoverySearch:
				for _, message := range publishQueue {
					controller.brokerConnection.SendMessage(message)
				}
			}
		}
	}()
}
