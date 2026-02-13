package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"strconv"
	"time"

	"mobile.dani.df/logging"
	upnp "mobile.dani.df/upnp-device"
)

func Ssdp(ctx context.Context) error {
	log := ctx.Value("logger").(logging.Logger)
	deviceXML := ""
	ctx = context.WithValue(ctx, "deviceXML", deviceXML)

	addr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		log.Error("[ssdp] Error while resolving address: " + err.Error())
		return errors.New("Resolve error")
	}
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)

	messageBuffer := make([]byte, 1024)
	for {
		n, source, err := conn.ReadFromUDP(messageBuffer)
		if err != nil {
			log.Error("[ssdp] Error while receiving a message")
		} else {
			message := UDPPacket{
				source:  *source,
				message: string(messageBuffer[:n]),
			}
			log.Info("[ssdp] Received message from " + message.source.String())

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
					log.Warn("[ssdp] Received a M-SEARCH without ST header")
				} else if err != nil && err.Error() == "Request not for this device" {
					log.Debug("[ssdp] Request not for this device")
				} else {
					for _, response := range responses {
						log.Debug("[ssdp] Responding to " + response.receiver.String() + " with " + response.message)

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
