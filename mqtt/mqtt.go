package mqtt

import (
	"context"
	"encoding/json"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"mobile.dani.df/logging"
)

type MqttConfig struct {
	MqttBroker string
}

type MqttConnection struct {
	client  mqtt.Client
	publish chan MqttMessage
	cancel  context.CancelFunc
}

type MqttMessage struct {
	Topic    string
	Qos      byte
	Retained bool
	Payload  string
}

func CreateConnection(ctx context.Context, config MqttConfig) (MqttConnection, error) {
	log := ctx.Value("logger").(logging.Logger)

	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(config.MqttBroker)
	mqttOpts.OnConnect = func(client mqtt.Client) {
		onConnectHandler(ctx)
	}
	mqttOpts.OnConnectionLost = func(client mqtt.Client, err error) {
		onConnectionErrorHandler(ctx, err)
	}

	client := mqtt.NewClient(mqttOpts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Error("[mqtt] Error while connecting to the broker: " + token.Error().Error())
		return MqttConnection{}, token.Error()
	}

	log.Info("[mqtt] Broker connection enstablished")

	ctx, cancel := context.WithCancel(ctx)
	conn := MqttConnection{
		client:  client,
		publish: make(chan MqttMessage, 128),
		cancel:  cancel,
	}

	publishDaemon(ctx, conn)

	return conn, nil
}

func TerminateConnection(conn MqttConnection) {
	conn.cancel()
	conn.client.Disconnect(50)
}

func (conn MqttConnection) SendMessage(topic string, qos byte, retained bool, payload string) {
	conn.publish <- MqttMessage{
		Topic:    topic,
		Qos:      qos,
		Retained: retained,
		Payload:  payload,
	}
}
func (conn MqttConnection) Subscribe(ctx context.Context, topic string, qos byte, handler func(MqttMessage)) error {
	log := ctx.Value("logger").(logging.Logger)

	subscribeHandler := func(client mqtt.Client, msg mqtt.Message) {
		log.Debug("[mqtt] {" + msg.Topic() + "}: <" + string(msg.Payload()) + ">")
		handler(MqttMessage{
			Topic:    msg.Topic(),
			Qos:      msg.Qos(),
			Retained: msg.Retained(),
			Payload:  string(msg.Payload()),
		})
	}

	token := conn.client.Subscribe(topic, qos, subscribeHandler)
	if token.Wait() && token.Error() != nil {
		log.Error("[mqtt] Error while subscribing to {" + topic + "}: " + token.Error().Error())
		return token.Error()
	}

	log.Info("[mqtt] Subscribe to {" + topic + "}")

	return nil
}

func (conn MqttConnection) Unsubscribe(ctx context.Context, topic string) error {
	log := ctx.Value("logger").(logging.Logger)

	token := conn.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		log.Error("[mqtt] Error while subscribing to {" + topic + "}: " + token.Error().Error())
		return token.Error()
	}

	return nil
}

func publishDaemon(ctx context.Context, conn MqttConnection) {
	go func() {
		log := ctx.Value("logger").(logging.Logger)

		for {
			select {
			case message := <-conn.publish:
				go func() {
					token := conn.client.Publish(message.Topic, message.Qos, message.Retained, message.Payload)
					token.Wait()
					if token.Error() != nil {
						log.Error("[mqtt] Error occured while publishing {" + message.Topic + "}: <" + message.Payload + ">")
					} else {
						log.Debug("[mqtt] Message published on topic {" + message.Topic + "}")
					}
				}()
			case <-ctx.Done():
				log.Info("[mqtt] Terminating publishing daemon")
				return
			}
		}
	}()
}

func onConnectHandler(ctx context.Context) {
	log := ctx.Value("logger").(logging.Logger)
	log.Info("[mqtt] Connected")
}

func onConnectionErrorHandler(ctx context.Context, err error) {
	log := ctx.Value("logger").(logging.Logger)
	log.Error("[mqtt] Connection error: " + err.Error())
}

func ParseDiscoveryMessage(message MqttMessage) Device {
	deviceType := strings.Split(message.Topic, "/")[1]

	result := Device{
		SwitchRootDevice: nil,
		SensorRootDevice: nil,
	}
	switch deviceType {
	case "switch":
		result.SwitchRootDevice = parseSwitchDeviceMessage(message.Payload)
	case "sensor":
		result.SensorRootDevice = parseSensorDeviceMessage(message.Payload)
	default:
		result.SwitchRootDevice = parseSwitchDeviceMessage(message.Payload)
	}

	return result
}

func parseSwitchDeviceMessage(message string) *SwitchRootDevice {
	result := SwitchRootDevice{}
	json.Unmarshal([]byte(message), &result)
	return &result
}

func parseSensorDeviceMessage(message string) *SensorRootDevice {
	result := SensorRootDevice{}
	json.Unmarshal([]byte(message), &result)
	return &result
}
