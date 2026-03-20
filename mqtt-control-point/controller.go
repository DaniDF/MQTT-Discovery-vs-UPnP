package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/DaniDF/MQTT-Discovery-vs-UPnP/logging"
	"github.com/DaniDF/MQTT-Discovery-vs-UPnP/mqtt"
	"github.com/DaniDF/MQTT-Discovery-vs-UPnP/utils"
)

const (
	mqttSearchTimeoutSeconds = 10
	mqttRetained             = false
)

type MqttController struct {
	mqttConfig           mqtt.MqttConfig
	ctx                  context.Context
	brokerConnection     mqtt.MqttConnection
	DiscoveryTopic       string
	AliveTopic           string
	Qos                  byte
	subscriptionChannels sync.Map
}

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

// Searches for mqtt devices
// Timeout: seconds to wait for messages, if <= 0 uses the default timeout of 10 seconds
func (controller *MqttController) Search(timeout int) []mqtt.Device {
	log := controller.ctx.Value("logger").(logging.Logger)

	result := []mqtt.Device{}

	if timeout <= 0 {
		timeout = mqttSearchTimeoutSeconds
	}

	wait := make(chan bool)

	handler := func(message mqtt.MqttMessage) {
		log.Debug("[mqtt-controller] Discovered: {" + message.Topic + "}: <" + message.Payload + ">")
		mqttDevice, err := mqtt.ParseDiscoveryMessage(message, controller.DiscoveryTopic)
		if err != nil {
			log.Warn("[mqtt-controller] Received not well formatted device discovery message: " + message.Payload + " topic: " + message.Topic + ". Error: " + err.Error())
			return
		}

		controller.subscriptionChannels.Store(mqttDevice.StateTopic, make(chan string, 128))

		err = controller.brokerConnection.Subscribe(controller.ctx, mqttDevice.StateTopic, controller.Qos, controller.listenSubscriptionHandler)
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
			stateChannel, found := controller.subscriptionChannels.Load(mqttDevice.StateTopic)
			if !found {
				log.Error("[mqtt-controller] Error while fetching state channel for " + mqttDevice.StateTopic)
				return "", errors.New("State channel not found")
			}
			state := <-stateChannel.(chan string)
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

	utils.AlertAfter(time.Duration(timeout)*time.Second, wait)

	<-wait

	return result
}

func (controller *MqttController) listenSubscriptionHandler(message mqtt.MqttMessage) {
	log := controller.ctx.Value("logger").(logging.Logger)

	subscriptionChannel, found := controller.subscriptionChannels.Load(message.Topic)
	if !found {
		log.Error("[mqtt-controller] Error while fetching subscription channel for " + message.Topic)
	} else {
		subscriptionChannel.(chan string) <- message.Payload
	}
}

var publishQueue = []mqtt.MqttMessage{}

func (controller *MqttController) PublishSwitchDevice(device *mqtt.Device) error {
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

	controller.subscriptionChannels.Store(device.CommandTopic, make(chan string))

	handler := func(message mqtt.MqttMessage) {
		commandChannel, found := controller.subscriptionChannels.Load(device.CommandTopic)
		if !found {
			log.Error("[mqtt-controller] Error while fetching command channel for " + device.CommandTopic)
		} else {
			commandChannel.(chan string) <- message.Payload
		}
	}

	controller.brokerConnection.Subscribe(controller.ctx, device.CommandTopic, byte(device.Qos), handler)

	device.GetRequiredState = func() string {
		commandChannel, found := controller.subscriptionChannels.Load(device.CommandTopic)
		if !found {
			log.Error("[mqtt-controller] Error while fetching command channel for " + device.CommandTopic)
			return ""
		}

		return <-commandChannel.(chan string)
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

	return nil
}

func (controller *MqttController) discoveryDaemon(qos byte) {
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
