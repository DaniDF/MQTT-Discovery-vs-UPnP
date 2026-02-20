package upnp

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/ipv4"
	"mobile.dani.df/logging"
	"mobile.dani.df/utils"
)

const (
	ssdpMulticastAddress               = "239.255.255.250"
	ssdpMulticastPort                  = 1900
	ssdpNotifyValiditySeconds          = 1800 // Seconds of validity for the NOTIFY message (see 1.2.2)
	ssdpWaitMillisBeforeSend           = 100  // Milliseconds between sends in NOTIFY
	ssdpMSearchMX                      = 2
	ssdpMSearchResponseValiditySeconds = 600
)

type MSearchResult struct {
	CacheControl int
	Date         time.Time
	Location     string
	Server       string
	St           string
	USN          string
}

// --------------------------------------------------------------------------------------
// For upnp control point
// --------------------------------------------------------------------------------------

func Search(ctx context.Context, st string) ([]MSearchResult, error) {
	log := ctx.Value("logger").(logging.Logger)

	addr, err := net.ResolveUDPAddr("udp4", ssdpMulticastAddress+":"+strconv.Itoa(ssdpMulticastPort))
	if err != nil {
		log.Error("[ssdp] Error while resolving address: " + err.Error())
		return []MSearchResult{}, errors.New("Resolve error")
	}

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		log.Error("[ssdp] Error while listen multicast UDP")
		return []MSearchResult{}, errors.New("Error listen multicast UDP")
	}
	defer conn.Close()

	message := generateSSDPMSearchMulticast(st, ssdpMSearchMX)

	packConn := ipv4.NewPacketConn(conn)
	err = packConn.SetMulticastTTL(2)
	if err != nil {
		log.Error("Error setting Multicast TTL: " + err.Error())
	}

	packConn.WriteTo([]byte(message.message), nil, addr)

	responses, err := listenMSearchResponse(ctx, conn, ssdpMSearchMX)
	if err != nil {
		return []MSearchResult{}, err
	}

	result := []MSearchResult{}
	for _, response := range responses {
		mResponse, err := parseMSearchResponse(response)
		if err == nil {
			result = append(result, mResponse)
		}
	}

	return result, nil
}

func listenMSearchResponse(ctx context.Context, conn *net.UDPConn, mx int) ([]string, error) {
	log := ctx.Value("logger").(logging.Logger)

	timeoutRead := make(chan bool, 1)
	utils.AlertAfter(time.Duration(mx+1)*time.Second, timeoutRead)

	responses := []string{}
	messageBuffer := make([]byte, 1024)
	for {
		select {
		case <-timeoutRead:
			log.Debug("[ssdp] Listen for M-Search responses ended by timeout")
			return responses, nil
		default:
			conn.SetReadDeadline(time.Now().Add(time.Duration(mx) * time.Second))
			n, source, err := conn.ReadFromUDP(messageBuffer)
			if err != nil {
				errorMessageSplit := strings.Split(err.Error(), ":")
				errorMessage := strings.TrimSpace(errorMessageSplit[len(errorMessageSplit)-1])

				if errorMessage != "i/o timeout" {
					log.Error("[ssdp] Error while receiving a message: " + err.Error())
					return responses, err
				}
			} else {
				responses = append(responses, string(messageBuffer[:n]))
				log.Debug("[ssdp] Received message from " + source.String())
			}
		}
	}
}

// Generates an UDPPacket for multicast M-Search as described in 1.3.2
func generateSSDPMSearchMulticast(st string, mx int) UDPPacket {
	return generateSSDPMSearch(st, mx, ssdpMulticastAddress, ssdpMulticastPort)
}

// Produces an UDPPacket for M-SEARCH as described in 1.3.2
func generateSSDPMSearch(st string, mx int, receiverAddr string, receiverPort int) UDPPacket {
	searchMessage := "M-SEARCH * HTTP/1.1\r\n" +
		"HOST: " + receiverAddr + ":" + strconv.Itoa(receiverPort) + "\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"MX: " + strconv.Itoa(mx) + "\r\n" +
		"ST: " + st + "\r\n" +
		"USER-AGENT: " + ServerUserAgent + "\r\n" +
		"\r\n"
	return UDPPacket{
		receiver: net.UDPAddr{
			IP:   net.ParseIP(ssdpMulticastAddress),
			Port: ssdpMulticastPort,
		},
		message: searchMessage,
	}
}

func parseMSearchResponse(response string) (MSearchResult, error) {
	cacheControl, find := FindHeader(response, "CACHE-CONTROL")
	if !find {
		return MSearchResult{}, errors.New("CACHE-CONTROL not present")
	}
	cacheControlMaxAge, err := strconv.Atoi(strings.TrimSpace(strings.Split(cacheControl, "=")[1]))
	if err != nil {
		return MSearchResult{}, errors.New("CACHE-CONTROL max-age not well formatted")
	}

	// Date in not "Required" but "Recommended" (see 1.3.3)
	dateString, find := FindHeader(response, "DATE")
	date := time.Now()
	if find {
		date, err = time.Parse(time.RFC1123, dateString)
		if err != nil {
			date = time.Now()
		}
	}

	location, find := FindHeader(response, "LOCATION")
	if !find {
		return MSearchResult{}, errors.New("LOCATION not present")
	}

	server, find := FindHeader(response, "SERVER")
	if !find {
		return MSearchResult{}, errors.New("SERVER not present")
	}

	st, find := FindHeader(response, "ST")
	if !find {
		return MSearchResult{}, errors.New("ST not present")
	}

	usn, find := FindHeader(response, "USN")
	if !find {
		return MSearchResult{}, errors.New("USN not present")
	}

	return MSearchResult{
		CacheControl: cacheControlMaxAge,
		Date:         date,
		Location:     location,
		Server:       server,
		St:           st,
		USN:          usn,
	}, nil
}

// --------------------------------------------------------------------------------------
// For upnp device
// --------------------------------------------------------------------------------------

func SsdpDevice(ctx context.Context, rootDevice RootDevice) error {
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
	defer conn.Close()

	log.Info("[ssdp] Listening for request")
	messageBuffer := make([]byte, 1024)
	for {
		n, source, err := conn.ReadFromUDP(messageBuffer)

		go func(message string, src net.UDPAddr, err error) {
			if err != nil {
				log.Error("[ssdp] Error while receiving a message")
				return
			}

			packet := UDPPacket{
				source:  *source,
				message: message,
			}
			log.Debug("[ssdp] Received message from " + packet.source.String())

			_, isMSearch := FindHeader(packet.message, "M-SEARCH")

			if isMSearch {
				log.Info("[ssdp] Received M-SEARCH from " + packet.source.String())
				// MX wait seconds to send the response to prevent DOS (see 1.3.2)
				wait := make(chan bool, 1)
				mx, findMx := FindHeader(packet.message, "MX")

				if findMx {
					mxValue, err := strconv.Atoi(mx)
					if err == nil {
						sleepTime := int((rand.Float32() * float32(mxValue)) * 1000)
						utils.AlertAfter(time.Duration(sleepTime)*time.Millisecond, wait)
					}
				} else {
					wait <- true
				}

				responses, err := handleSSDPMSEARCHRequest(packet, rootDevice)

				if err != nil && err.Error() == "Request not valid: ST not present" {
					log.Warn("[ssdp] Received a M-SEARCH without ST header")
				} else if err != nil && err.Error() == "Request not for this device" {
					log.Debug("[ssdp] Request not for this device")
				} else {
					<-wait // Fun fun fact: my tvs never wait and reply immediately
					for _, response := range responses {
						log.Debug("[ssdp] Responding to " + response.receiver.String() + " with " + response.message)
						conn.WriteToUDP([]byte(response.message), &response.receiver)
					}
				}
			} else {
				log.Debug("[ssdp] NOT M-SEARCH Received message from " + packet.source.String())
			}

		}(string(messageBuffer[:n]), *source, err)
	}
}

// Handles a single SSDP request
// Returns error if it is not a M-SEARCH request
func handleSSDPMSEARCHRequest(message UDPPacket, rootDevice RootDevice) ([]UDPPacket, error) {
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
	genByServiceType := func(service Service) UDPPacket {
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

// Produces an UDPPacket for responding to M-SEARCH as described in 1.3.3
func generateSSDPResponseByDevice(st string, usn string, device Device, request UDPPacket) UDPPacket {
	responseMessage := "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: max-age = " + strconv.Itoa(ssdpMSearchResponseValiditySeconds) + "\r\n" +
		"DATE: " + time.Now().Format(time.RFC1123) + "\r\n" +
		"EXT:\r\n" +
		"LOCATION: " + device.PresentationURL + "\r\n" +
		"SERVER: " + ServerUserAgent + "\r\n" +
		"ST: " + st + "\r\n" +
		"USN: " + usn + "\r\n" +
		"\r\n"
	return UDPPacket{
		receiver: request.source,
		message:  responseMessage,
	}
}

// Runs the daemon that periodically multicasts the NOTIFY message
func ssdpNotifyDaemon(ctx context.Context, addr *net.UDPAddr, rootDevice RootDevice) {
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
func generateSSDPNotifyMessage(rootDevice RootDevice) []UDPPacket {
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
func generateSSDPNotifyMessageForRootDevice(rootDevice RootDevice) UDPPacket {
	nt := "upnp:rootdevice"
	usn := rootDevice.Device.UDN + "::upnp:rootdevice"

	return generateSSDPNotifyMessageByDevice(nt, usn, rootDevice.Device)
}

// Produces two distinct UDPPacket as described in 1.2.2 Table 1-1 and Table 1-2
func generateSSDPNotifyMessageForDevice(device Device) (UDPPacket, UDPPacket) {
	nt1 := device.UDN
	usn1 := nt1

	nt2 := device.DeviceType
	usn2 := device.UDN + "::" + device.DeviceType

	return generateSSDPNotifyMessageByDevice(nt1, usn1, device), generateSSDPNotifyMessageByDevice(nt2, usn2, device)
}

// Produces two distinct UDPPacket as described in 1.2.2 Table 1-3
func generateSSDPNotifyMessageForService(device Device, service Service) UDPPacket {
	nt1 := service.ServiceType
	usn1 := device.UDN + "::" + service.ServiceType

	return generateSSDPNotifyMessageByDevice(nt1, usn1, device)
}

// Generates the UDPPacket formatted for NOTIFY
func generateSSDPNotifyMessageByDevice(nt string, usn string, device Device) UDPPacket {
	responseMessage := "NOTIFY * HTTP/1.1\r\n" +
		"HOST: " + ssdpMulticastAddress + ":" + strconv.Itoa(ssdpMulticastPort) + "\r\n" +
		"CACHE-CONTROL: max-age = " + strconv.Itoa(ssdpNotifyValiditySeconds) + "\r\n" +
		"LOCATION: " + device.PresentationURL + "\r\n" +
		"NT: " + nt + "\r\n" +
		"NTS: ssdp:alive\r\n" +
		"SERVER: " + ServerUserAgent + "\r\n" +
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
