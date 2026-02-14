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

const (
	ssdpMulticastAddress      = "239.255.255.250"
	ssdpMulticastPort         = 1900
	ssdpNotifyValiditySeconds = 1800 // Seconds of validity for the NOTIFY message (see 1.2.2)
	ssdpWaitMillisBeforeSend  = 100  // Milliseconds between sends in NOTIFY
)

func Ssdp(ctx context.Context) error {
	log := ctx.Value("logger").(logging.Logger)
	deviceXML := ""
	ctx = context.WithValue(ctx, "deviceXML", deviceXML)

	addr, err := net.ResolveUDPAddr("udp4", ssdpMulticastAddress+":"+strconv.Itoa(ssdpMulticastPort))
	if err != nil {
		log.Error("[ssdp] Error while resolving address: " + err.Error())
		return errors.New("Resolve error")
	}

	ssdpNotifyDaemon(ctx, addr, rootDevice)

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		log.Error("[ssdp] Error while listen multicast UDP")
		return errors.New("Error while listen")
	}

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

				// MX wait seconds to send the response to prevent DOS (see 1.3.2)
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

				responses, err := handleSSDPRequest(message, rootDevice)

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

// Handles a single SSDP request
// Returns error if it is not a M-SEARCH request
func handleSSDPRequest(message UDPPacket, rootDevice upnp.RootDevice) ([]UDPPacket, error) {
	fmt.Println(message.source.IP.String() + ": " + message.message) //TODO remove

	st, findSt := FindHeader(message.message, "ST")
	if !findSt {
		return []UDPPacket{}, errors.New("Request not valid: ST not present")
	}

	genByDeviceUUID := func() UDPPacket {
		usn := rootDevice.Device.UDN
		return generateSSDPResponseByDevice(st, usn, rootDevice.Device, message)
	}
	genByRootDevice := func() UDPPacket {
		usn := rootDevice.Device.UDN + "::upnp:rootdevice"
		return generateSSDPResponseByDevice(st, usn, rootDevice.Device, message)
	}
	genByDeviceType := func() UDPPacket {
		usn := rootDevice.Device.UDN + "::" + rootDevice.Device.DeviceType
		return generateSSDPResponseByDevice(st, usn, rootDevice.Device, message)
	}
	genByServiceType := func(service upnp.Service) UDPPacket {
		usn := rootDevice.Device.UDN + "::" + service.ServiceType
		return generateSSDPResponseByDevice(st, usn, rootDevice.Device, message)
	}

	result := []UDPPacket{}

	if st == "ssdp:all" {
		result = append(result, genByDeviceUUID(), genByRootDevice(), genByDeviceType())
		for _, service := range rootDevice.Device.ServiceList {
			result = append(result, genByServiceType(service))
		}
	} else {
		switch st {
		case rootDevice.Device.UDN:
			result = append(result, genByDeviceUUID())

		case "upnp:rootdevice":
			result = append(result, genByRootDevice())

		case rootDevice.Device.DeviceType:
			result = append(result, genByDeviceType())

		default:
			for _, service := range rootDevice.Device.ServiceList {
				if st == service.ServiceType {
					result = append(result, genByServiceType(service))
				}
			}
		}
	}

	if len(result) == 0 {
		return []UDPPacket{}, errors.New("Request not for this device")
	}
	return result, nil
}

// Produces an UDPPacket as described in 1.3.3
func generateSSDPResponseByDevice(st string, usn string, device upnp.Device, request UDPPacket) UDPPacket {
	responseMessage := "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: max-age = 120\r\n" +
		//TODO Add date
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

// Runs the daemon that periodically multicasts the NOTIFY message
func ssdpNotifyDaemon(ctx context.Context, addr *net.UDPAddr, rootDevice upnp.RootDevice) {
	go func() {
		log := ctx.Value("logger").(logging.Logger)

		notify := func() {
			conn, err := net.DialUDP("udp4", nil, addr)
			if err != nil {
				log.Error("[ssdp] Error while dial UDP")
			} else {
				for _, message := range generateSSDPNotifyMessage(rootDevice) {
					conn.Write([]byte(message.message))
					time.Sleep(ssdpWaitMillisBeforeSend * time.Millisecond)
				}
			}
		}

		notify()

		flagFinish := false
		for !flagFinish {
			select {
			case <-ctx.Done():
				flagFinish = true
			case <-time.After(ssdpNotifyValiditySeconds / 2 * time.Second): // Re-notify again after half CACHE-CONTROL: max-age of the NOTIFY See 1.2.2
				notify()
			}
		}
	}()
}

// Generates the list of packets to be send during a NOTIFY
func generateSSDPNotifyMessage(rootDevice upnp.RootDevice) []UDPPacket {
	result := []UDPPacket{}

	// RootDevice 3 messages
	result = append(result, generateSSDPNotifyMessageForRootDevice(rootDevice))
	secondRootMessage, thirdRootMessage := generateSSDPNotifyMessageForDevice(rootDevice.Device)
	result = append(result, secondRootMessage, thirdRootMessage)

	// EmbeddedDevices 2 messages
	for _, embeddedDevice := range rootDevice.Device.EmbeddedDevices {
		firstDeviceMessage, secondDeviceMessage := generateSSDPNotifyMessageForDevice(embeddedDevice)
		result = append(result, firstDeviceMessage, secondDeviceMessage)
	}

	for _, service := range rootDevice.Device.ServiceList {
		result = append(result, generateSSDPNotifyMessageForService(rootDevice.Device, service))
	}
	for _, embeddedDevice := range rootDevice.Device.EmbeddedDevices {
		for _, embeddedDeviceService := range embeddedDevice.ServiceList {
			result = append(result, generateSSDPNotifyMessageForService(embeddedDevice, embeddedDeviceService))
		}
	}

	return result
}

// Produces an UDPPacket as described in 1.2.2 Table 1-1
func generateSSDPNotifyMessageForRootDevice(rootDevice upnp.RootDevice) UDPPacket {
	nt := "upnp:rootdevice"
	usn := "uuid:" + rootDevice.Device.UDN + "::upnp:rootdevice"

	return generateSSDPNotifyMessageByDevice(nt, usn, rootDevice.Device)
}

// Produces two distinct UDPPacket as described in 1.2.2 Table 1-1 and Table 1-2
func generateSSDPNotifyMessageForDevice(device upnp.Device) (UDPPacket, UDPPacket) {
	nt1 := "uuid:" + device.UDN
	usn1 := nt1

	nt2 := device.DeviceType
	usn2 := "uuid:" + device.UDN + "::" + device.DeviceType

	return generateSSDPNotifyMessageByDevice(nt1, usn1, device), generateSSDPNotifyMessageByDevice(nt2, usn2, rootDevice.Device)
}

// Produces two distinct UDPPacket as described in 1.2.2 Table 1-3
func generateSSDPNotifyMessageForService(device upnp.Device, service upnp.Service) UDPPacket {
	nt1 := service.ServiceType
	usn1 := "uuid:" + device.UDN + "::" + service.ServiceType

	return generateSSDPNotifyMessageByDevice(nt1, usn1, device)
}

// Generates the UDPPacket formatted for NOTIFY
func generateSSDPNotifyMessageByDevice(nt string, usn string, device upnp.Device) UDPPacket {
	responseMessage := "NOTIFY * HTTP/1.1\r\n" +
		"HOST: " + ssdpMulticastAddress + ":" + strconv.Itoa(ssdpMulticastPort) + "\r\n" +
		"CACHE-CONTROL: max-age = " + strconv.Itoa(ssdpNotifyValiditySeconds) + "\r\n" +
		"LOCATION: " + device.PresentationURL + "\r\n" +
		"NT: " + nt + "\r\n" +
		"NTS: ssdp:alive\r\n" +
		"SERVER: DFOS/0.1 UPnP/2.0 123/1.1\r\n" +
		"USN: " + usn + "\r\n" +
		"\r\n"
	return UDPPacket{
		receiver: net.UDPAddr{
			IP:   net.ParseIP(ssdpMulticastAddress),
			Port: ssdpMulticastPort,
		},
		message: responseMessage,
	}
}
