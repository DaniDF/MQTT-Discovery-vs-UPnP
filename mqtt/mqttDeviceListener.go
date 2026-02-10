package mqtt

import (
	"errors"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/net/context"
	"mobile.dani.df/logging"
)

type MqttConfig struct {
	MqttBroker string
}

type MqttMessage struct {
	Topic    string
	Qos      byte
	Retained bool
	Payload  string
}

type Connection struct {
	log    logging.Logger
	client mqtt.Client
	//done    chan bool
	publish chan MqttMessage
	ctx     context.Context
	cancel  context.CancelFunc
}

func (conn Connection) SendMessage(topic string, qos byte, retained bool, payload string) {
	conn.publish <- MqttMessage{
		Topic:    topic,
		Qos:      qos,
		Retained: retained,
		Payload:  payload,
	}
}
func (conn Connection) Subscribe(topic string, qos byte, handler func(MqttMessage)) {
	subscribeHandler := func(client mqtt.Client, msg mqtt.Message) {
		conn.log.Debug("[" + msg.Topic() + "]: <" + string(msg.Payload()) + ">")
		handler(MqttMessage{
			Topic:    msg.Topic(),
			Qos:      msg.Qos(),
			Retained: msg.Retained(),
			Payload:  string(msg.Payload()),
		})
	}
	conn.client.Subscribe(topic, qos, subscribeHandler)
}
func (conn Connection) Unsubscribe(topic string) {
	conn.client.Unsubscribe(topic)
}

func CreateConnection(ctx context.Context, config MqttConfig) (Connection, error) {
	log := ctx.Value("logger").(logging.Logger)

	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(config.MqttBroker)
	mqttOpts.OnConnect = func(client mqtt.Client) {
		onConnectHandler(client, log)
	}
	mqttOpts.OnConnectionLost = func(client mqtt.Client, err error) {
		onConnectionErrorHandler(client, log, err)
	}

	client := mqtt.NewClient(mqttOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Error("Error while connecting to the broker: " + token.Error().Error())

		return Connection{}, errors.New("Connection error")
	} else {
		log.Debug("Broker connection enstablished")
	}

	ctx, cancel := context.WithCancel(ctx)
	conn := Connection{
		log:     log,
		client:  client,
		publish: make(chan MqttMessage),
		ctx:     ctx,
		cancel:  cancel,
	}

	publishDaemon(conn)

	return conn, nil
}

func TerminateConnection(conn Connection) {
	conn.cancel()
	conn.client.Disconnect(50)
}

func publishDaemon(conn Connection) {
	go func() {
		log := conn.ctx.Value("logger").(logging.Logger)
		flagStop := false
		for !flagStop {
			select {
			case message := <-conn.publish:
				token := conn.client.Publish(message.Topic, message.Qos, message.Retained, message.Payload)

				go func() {
					token.Wait()
					if token.Error() != nil {
						log.Error("Error occured while publishing")
					} else {
						log.Debug("Message published")
					}
				}()
			case <-conn.ctx.Done():
				log.Debug("Terminating publishing daemon")
				flagStop = true
			}
		}
	}()
}

func onConnectHandler(client mqtt.Client, log logging.Logger) {
	log.Info("Connected!")
}

func onConnectionErrorHandler(client mqtt.Client, log logging.Logger, err error) {
	log.Error("Connection error: " + err.Error())
}
