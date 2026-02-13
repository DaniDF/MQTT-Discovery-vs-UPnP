package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"time"

	"mobile.dani.df/logging"
	upnp "mobile.dani.df/upnp-device"
)

const (
	upnpPort              = 8080
	devicePresentationUrl = "/Device.xml"
)

var rootDevice = upnp.RootDevice{
	SpecVersion: upnp.SpecVersion{
		Major: "0",
		Minor: "1",
	},
	Devices: []upnp.Device{
		{
			DeviceType:       "urn:schemas-upnp-org:device:BinaryLight:1",
			UDN:              "uuid:55076f6e-6b79-4d65-6401-00d0b811d10b", //TODO to be generate random (see specification 1.1.4)
			FriendlyName:     "SmartLight",
			Manufacturer:     "DF Corp.",
			ManufacturerURL:  "http://superlight.df",
			ModelName:        "SmartLight pro plus",
			ModelURL:         "http://superlight.df/smartlight-pro-plus",
			ModelDescription: "The best smart light",
			ModelNumber:      "422",
			SerialNumber:     "123 - 456 - 789 - 0",
			UPC:              "12345678900987654321",
			PresentationURL:  "http://" + GetLocalIP() + ":" + strconv.Itoa(upnpPort) + devicePresentationUrl,
			IconList: []upnp.Icon{
				{
					Mimetype: "image/jpeg",
					Height:   "48",
					Width:    "48",
					Depth:    "24",
					Url:      "/images/icon-48x48.jpg",
				},
				{
					Mimetype: "image/jpeg",
					Height:   "120",
					Width:    "120",
					Depth:    "24",
					Url:      "/images/icon-120x120.jpg",
				},
			},
			ServiceList: []upnp.Service{
				{
					ServiceType: "urn:schemas-upnp-org:service:SwitchPower:1",
					ServiceId:   "urn:upnp-org:serviceId:SwitchPower",
					SCPDURL:     "/SwitchPower",
					EventSubURL: "/SwitchPower/event",
					ControlURL:  "/SwitchPower/control",
				},
				{
					ServiceType: "urn:schemas-upnp-org:service:TemperatureSensor:1",
					ServiceId:   "urn:upnp-org:serviceId:TemperatureSensor",
					SCPDURL:     "/TemperatureSensor",
					EventSubURL: "/TemperatureSensor/event",
					ControlURL:  "/TemperatureSensor/control",
				},
			},
		},
	},
}

type UDPPacket struct {
	source   net.UDPAddr
	receiver net.UDPAddr
	message  string
}

func (m UDPPacket) String() string {
	return m.source.String() + " says " + m.message
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ctx = logging.Init(ctx)

	httpServer(ctx)
	ssdp(ctx)

	cancel() //TODO Find a solution it is unused
}

func ssdp(ctx context.Context) error {
	log := ctx.Value("logger").(logging.Logger)
	deviceXML := ""
	ctx = context.WithValue(ctx, "deviceXML", deviceXML)

	addr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		log.Error("Error while resolving address: " + err.Error())
		return errors.New("Resolve error")
	}
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)

	messageBuffer := make([]byte, 1024)
	for {
		n, source, err := conn.ReadFromUDP(messageBuffer)
		if err != nil {
			log.Error("Error while receiving a message")
		} else {
			message := UDPPacket{
				source:  *source,
				message: string(messageBuffer[:n]),
			}
			log.Info("Received message from " + message.source.String())

			_, isMSearch := FindHeader(message.message, "M-SEARCH")

			if isMSearch {
				wait := make(chan bool)

				mx, findMx := FindHeader(message.message, "MX")

				if findMx {
					mxValue, err := strconv.Atoi(mx)
					if err == nil {
						go func() {
							sleepTime := int((rand.Float32() * float32(mxValue)) * 1000)
							time.Sleep(time.Duration(sleepTime) * time.Millisecond)
							wait <- true
						}()
					}
				} else {
					wait <- true
				}

				responses, err := handleSSDPRequest(message)

				if err != nil && err.Error() == "Request not valid: ST not present" {
					log.Warn("Received a M-SEARCH without ST header")
				} else if err != nil && err.Error() == "Request not for this device" {
					log.Debug("Request not for this device")
				} else {
					for _, response := range responses {
						log.Debug("Responding to " + response.receiver.String() + " with " + response.message)

						<-wait
						conn.WriteToUDP([]byte(response.message), &response.receiver)
					}
				}
			}
		}
	}
}

func handleSSDPRequest(message UDPPacket) ([]UDPPacket, error) {
	fmt.Println(message.source.IP.String() + ": " + message.message) //TODO remove

	st, findSt := FindHeader(message.message, "ST")
	if !findSt {
		return []UDPPacket{}, errors.New("Request not valid: ST not present")
	}

	result := []UDPPacket{}
	usn := ""

	if st == "ssdp:all" {
		for _, device := range rootDevice.Devices {
			usn = device.UDN
			result = append(result, generateSSDPResponseByDevice(st, usn, device, message))
		}
	} else {
		for _, device := range rootDevice.Devices {
			switch st {
			case device.UDN:
				usn = device.UDN
				result = append(result, generateSSDPResponseByDevice(st, usn, device, message))

			case "upnp:rootdevice":
				usn = device.UDN + "::upnp:rootdevice"
				result = append(result, generateSSDPResponseByDevice(st, usn, device, message))

			case device.DeviceType:
				usn = device.UDN + "::" + device.DeviceType
				result = append(result, generateSSDPResponseByDevice(st, usn, device, message))

			default:
				for _, service := range device.ServiceList {
					if st == service.ServiceType {
						usn = device.UDN + "::" + service.ServiceType
						result = append(result, generateSSDPResponseByDevice(st, usn, device, message))
					}
				}
			}
		}
	}

	if len(result) == 0 {
		return []UDPPacket{}, errors.New("Request not for this device")
	}
	return result, nil
}

func generateSSDPResponseByDevice(st string, usn string, device upnp.Device, request UDPPacket) UDPPacket {
	responseMessage := "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: max-age = 2\r\n" +
		//Add date
		"EXT:\r\n" +
		"LOCATION: " + device.PresentationURL + "\r\n" +
		"SERVER: DFOS/0.1 UPnP/2.0 123/1.1\r\n" +
		"ST: " + st + "\r\n" +
		"USN: " + usn + "\r\n" +
		"\r\n"
	return UDPPacket{
		receiver: request.source,
		message:  responseMessage,
	}
}

func httpServer(ctx context.Context) {
	go func() {
		log := ctx.Value("logger").(logging.Logger)

		http.HandleFunc(devicePresentationUrl, func(resp http.ResponseWriter, req *http.Request) { deviceDescriptionHandler(ctx, resp, req) })

		log.Info("Listening for request")
		err := http.ListenAndServe(GetLocalIP()+":8080", nil)
		log.ErrorContext(ctx, "Error occurred while listen and serve: "+err.Error())
	}()
}

func deviceDescriptionHandler(ctx context.Context, response http.ResponseWriter, request *http.Request) {
	log := ctx.Value("logger").(logging.Logger)

	log.Info("Request from " + request.RemoteAddr + " resource " + request.RequestURI)

	response.WriteHeader(200)
	response.Write([]byte("TESTTTTT"))
}
