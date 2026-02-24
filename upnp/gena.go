package upnp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"mobile.dani.df/device-service"
	"mobile.dani.df/logging"
	"mobile.dani.df/utils"
)

const (
	genaSubscriptionTimeoutSeconds   = 1800
	genaMulticastNotificationAddress = "239.255.255.246"
	genaMulticastNotificationPort    = 7900
)

type GenaMulticastEventLevels string

const (
	Emergency GenaMulticastEventLevels = "upnp:/emergency"
	Fault     GenaMulticastEventLevels = "upnp:/fault"
	Warning   GenaMulticastEventLevels = "upnp:/warning"
	Info      GenaMulticastEventLevels = "upnp:/info"
	Debug     GenaMulticastEventLevels = "upnp:/debug"
	General   GenaMulticastEventLevels = "upnp:/general"
)

func (level GenaMulticastEventLevels) String() string {
	switch level {
	case Emergency:
		return "upnp:/emergency"
	case Fault:
		return "upnp:/fault"
	case Warning:
		return "upnp:/warning"
	case Info:
		return "upnp:/info"
	case Debug:
		return "upnp:/debug"
	case General:
		return "upnp:/general"
	default:
		return "upnp:/general"
	}
}

type subscriptionRequest struct {
	sid       string
	userAgent string
	callback  *url.URL
	nt        string
	timeout   int
	statevar  []string
}

func (request subscriptionRequest) String() string {
	var result strings.Builder

	result.WriteString("USER-AGENT: " + request.userAgent + "\n")
	result.WriteString("CALLBACK: " + request.callback.String() + "\n")
	result.WriteString("NT: " + request.nt + "\n")
	result.WriteString("TIMEOUT: " + strconv.Itoa(request.timeout) + "\n")
	result.WriteString("STATEVAR: [" + utils.StringToCSV(request.statevar) + "]")

	return result.String()
}

// See 4.1.1
type subscriber struct {
	eventKey             int
	httpSuppertedVersion string
	userAgent            string
}

type subscription struct {
	sid        int64
	subscriber subscriber
	service    Service
	stateVar   []string
	creation   time.Time
	timeout    int
	callback   *url.URL
}

func (subscription subscription) Equal(other subscription) bool {
	return subscription.sid == other.sid
}

type stateVariableValue struct {
	stateVar *StateVariable
	value    string
}

type notification struct {
	service             Service
	stateVariableValues []stateVariableValue
}

var insertUpdateSubscription = make(chan subscription, 128)
var deleteSubscription = make(chan int64, 128)
var notificationStateChange = make(chan notification, 128)
var subscriptionsDB = make(map[int64]subscription)          //TODO Also consider sync.Map
var serviceSubscriptionDB = make(map[string][]subscription) //TODO Also consider sync.Map

// --------------------------------------------------------------------------------------
// For upnp control point
// --------------------------------------------------------------------------------------

func GenaSubscribeToService(ctx context.Context, rootDevice RootDevice, service Service, handler func(string), stateVars ...string) (*context.CancelFunc, string, error) {
	log := ctx.Value("logger").(logging.Logger)

	listenCtx, cancel := context.WithCancel(ctx)
	//_, err := listenAtMulticast(listenCtx, genaMulticastNotificationAddress, genaMulticastNotificationPort, func(ctx context.Context, p UDPPacket) { genaSubscriptionEventHandler(ctx, p, handler) })
	addr, err := listenAt(listenCtx, 0, func(ctx context.Context, p TCPPacket) { genaSubscriptionEventHandler(ctx, p, handler) })
	if err != nil {
		log.Error("[gena] An error occurred while listening for events: " + err.Error())
		cancel()
		return nil, "", err
	}
	log.Info("[gena] Start listening for subscription messages at " + addr.String())

	var rootUrl *url.URL
	if len(rootDevice.Device.PresentationURL) > 0 {
		if rootDevice.Device.PresentationURL[len(rootDevice.Device.PresentationURL)-1] == '/' {
			rootDevice.Device.PresentationURL = rootDevice.Device.PresentationURL[:len(rootDevice.Device.PresentationURL)-1]
		}
		rootUrl, err = url.Parse(rootDevice.Device.PresentationURL)

		if err != nil {
			log.Warn("[gena] An error occurred while parsing presetation url (upnp 1.1): " + err.Error())
			rootUrl = nil
		}
	}

	// URLBase as fallback
	if rootUrl == nil {
		if len(rootDevice.URLBase) > 0 {
			if rootDevice.URLBase[len(rootDevice.URLBase)-1] == '/' {
				rootDevice.URLBase = rootDevice.URLBase[:len(rootDevice.URLBase)-1]
			}
			rootUrl, err = url.Parse(rootDevice.URLBase + service.EventSubURL)

			if err != nil {
				log.Error("[gena] An error occurred while parsing URLBase (upnp <1.1): " + err.Error())
				cancel()
				return nil, "", err
			}
		} else {
			log.Error("[gena] Nor presentation url neither URLBase (upnp <= 1.1) are valid")
			cancel()
			return nil, "", errors.New("Device without valid url")
		}
	}

	subscriptionUrl, _ := url.Parse(rootUrl.Scheme + "://" + rootUrl.Host + service.EventSubURL)

	log.Debug("[gena] Attempting subscription at: " + subscriptionUrl.String())

	subscriptionRequest, err := http.NewRequest("SUBSCRIBE", subscriptionUrl.String(), nil)
	if err != nil {
		log.Error("[gena] An error occurred while creating a new request: " + err.Error())
		cancel()
		return nil, "", err
	}

	callbackUrl := "http://" + utils.GetLocalIP() + ":" + strconv.Itoa(addr.Port)

	subscriptionRequest.Header.Set("HOST", subscriptionUrl.Host)
	subscriptionRequest.Header.Set("USER-AGENT", ClientUserAgent)
	subscriptionRequest.Header.Set("CALLBACK", "<"+callbackUrl+">")
	subscriptionRequest.Header.Set("NT", "upnp:event")
	subscriptionRequest.Header.Set("TIMEOUT", "Second-"+strconv.Itoa(genaSubscriptionTimeoutSeconds))

	// STATEVARS is recommended not required (see 4.1.2)
	if len(stateVars) > 0 {
		var stateVarCSV strings.Builder
		for i, stateVar := range stateVars {
			if i != 0 {
				stateVarCSV.WriteString(",")
			}
			stateVarCSV.WriteString(stateVar)
		}
		subscriptionRequest.Header.Set("STATEVAR", stateVarCSV.String())
	}

	httpClient := &http.Client{
		Timeout: 3 * time.Second,
	}

	subscriptionResponse, err := httpClient.Do(subscriptionRequest)
	if err != nil {
		log.Error("[gena] Error while sending subscription request: " + err.Error())
		cancel()
		return nil, "", err
	}

	if subscriptionResponse.StatusCode != 200 {
		log.Error("[gena] Subscription returned with code: " + subscriptionResponse.Status)
		cancel()
		return nil, "", errors.New(subscriptionResponse.Status)
	}

	sid := subscriptionResponse.Header.Get("SID")

	log.Info("[gena] Subscription returned with code: " + subscriptionResponse.Status + " - sid: " + string(sid))

	return &cancel, sid, nil
}

func genaSubscriptionEventHandler(ctx context.Context, packet TCPPacket, handler func(string)) {
	log := ctx.Value("logger").(logging.Logger)

	log.Debug("[gena] Received from " + packet.source.String() + " subscription message: " + packet.message)

	//TODO parse the message and give only the values
	handler(packet.message)
}

// Listen to a specified port for UDP connection. When a client connects the handler function is invoked.
// If port = 0 is selected a random port number.
func listenAt(ctx context.Context, port int, handler func(context.Context, TCPPacket)) (*net.TCPAddr, error) {
	log := ctx.Value("logger").(logging.Logger)

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		log.Error("[gena] Error while listening TCP packet")
		return nil, err
	}

	listenAtDaemon(ctx, listener, handler)

	return listener.Addr().(*net.TCPAddr), nil
}

//TODO It has to listen for udp
/*func listenAtMulticast(ctx context.Context, multicastAddress string, port int, handler func(context.Context, UDPPacket)) (*net.UDPAddr, error) {
	log := ctx.Value("logger").(logging.Logger)

	addr, err := net.ResolveUDPAddr("udp4", multicastAddress+":"+strconv.Itoa(port))
	if err != nil {
		log.Error("[gena] Error while resolving address: " + err.Error())
		return nil, errors.New("Resolve error")
	}

	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		log.Error("[gena] Error while listening UDP packet")
		return nil, err
	}

	listenAtDaemon(ctx, conn, handler)

	return addr, nil
}
*/

func listenAtDaemon(ctx context.Context, listener *net.TCPListener, handler func(context.Context, TCPPacket)) {
	go func() {
		log := ctx.Value("logger").(logging.Logger)

		defer listener.Close()

		for {
			conn, err := listener.Accept() //conn.ReadFromUDP(messageBuffer)
			if err != nil {
				log.Error("[gena] Error while receiving TCP packet")
			}

			go func() {
				defer conn.Close()

				messageBuffer := make([]byte, 1024)
				n, err := conn.Read(messageBuffer)
				if err != nil {
					log.Error("[gena] Error while reading from connection: " + err.Error())
					return
				}

				go func() {
					handler(ctx, TCPPacket{
						source:  *conn.RemoteAddr().(*net.TCPAddr),
						message: string(messageBuffer[:n]),
					})
				}()

				_, err = conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
				if err != nil {
					log.Error("[gena] Error while writing the notification response: " + err.Error())
					return
				}
			}()

		}
	}()
}

func GenaUnsubscribeFromService(ctx context.Context, rootDevice RootDevice, service Service, sid string) error {
	log := ctx.Value("logger").(logging.Logger)

	var rootUrl *url.URL
	var err error
	if len(rootDevice.Device.PresentationURL) > 0 {
		if rootDevice.Device.PresentationURL[len(rootDevice.Device.PresentationURL)-1] == '/' {
			rootDevice.Device.PresentationURL = rootDevice.Device.PresentationURL[:len(rootDevice.Device.PresentationURL)-1]
		}

		rootUrl, err = url.Parse(rootDevice.Device.PresentationURL)

		if err != nil {
			log.Warn("[gena] An error occurred while parsing presetation url (upnp 1.1): " + err.Error())
			rootUrl = nil
		}
	}

	// URLBase as fallback
	if rootUrl == nil {
		if len(rootDevice.URLBase) > 0 {
			if rootDevice.URLBase[len(rootDevice.URLBase)-1] == '/' {
				rootDevice.URLBase = rootDevice.URLBase[:len(rootDevice.URLBase)-1]
			}
			rootUrl, err = url.Parse(rootDevice.URLBase + service.EventSubURL)

			if err != nil {
				log.Error("[gena] An error occurred while parsing URLBase (upnp <1.1): " + err.Error())
				return err
			}
		} else {
			log.Error("[gena] Nor presentation url neither URLBase (upnp <= 1.1) are valid")
			return errors.New("Device without valid url")
		}
	}

	unsubscriptionUrl, _ := url.Parse(rootUrl.Scheme + "://" + rootUrl.Host + service.EventSubURL)

	log.Debug("[gena] Attempting unsubscription at: " + unsubscriptionUrl.String())

	unsubscriptionRequest, err := http.NewRequest("UNSUBSCRIBE", unsubscriptionUrl.String(), nil)
	if err != nil {
		log.Error("[gena] An error occurred while creating a new request: " + err.Error())
		return err
	}

	unsubscriptionRequest.Header.Set("HOST", unsubscriptionUrl.Host)
	unsubscriptionRequest.Header.Set("SID", sid)

	httpClient := &http.Client{
		Timeout: 3 * time.Second,
	}

	unsubscriptionResponse, err := httpClient.Do(unsubscriptionRequest)
	if err != nil {
		log.Error("[gena] Error while sending unsubscription request: " + err.Error())
		return err
	}
	defer unsubscriptionResponse.Body.Close()

	if unsubscriptionResponse.StatusCode != 200 {
		log.Error("[gena] Unsubscription returned with code: " + unsubscriptionResponse.Status)
		return errors.New(unsubscriptionResponse.Status)
	}

	log.Info("[gena] Unubscription returned with code: " + unsubscriptionResponse.Status)

	return nil
}

// --------------------------------------------------------------------------------------
// For upnp device
// --------------------------------------------------------------------------------------

func GenaSubscriptionHandler(ctx context.Context, service Service, request *http.Request, response http.ResponseWriter) error {
	log := ctx.Value("logger").(logging.Logger)

	subscriptionRequest, err := parseSubscriptionRequest(ctx, request)
	if err != nil {
		switch { // See 4.1.2 table 4-4
		case err.Error() == "Callback paring error":
			generateNegativeResponse(412, response)
			return err
		default:
			generateNegativeResponse(400, response)
			return err
		}
	}

	var sid string
	if subscriptionRequest.sid != "" && subscriptionRequest.nt == "" && subscriptionRequest.callback == nil { // Subscription update
		sid, err = createNewSubscription(subscriptionRequest, service)

	} else if subscriptionRequest.sid == "" && subscriptionRequest.nt == "upnp:event" && subscriptionRequest.callback != nil { //New subscription
		sid, err = createNewSubscription(subscriptionRequest, service)

	} else { // Error invalid combination
		log.Warn("[gena] Received invalid subscription message: invalid combination of SID, NT, CALLBACK")
		generateNegativeResponse(400, response)
		return errors.New("Invalid combination of SID, NT, CALLBACK")
	}

	log.Debug("[gena] Received subscription with message: " + subscriptionRequest.String())

	generatePositiveResponse(subscriptionRequest, sid, response)

	return nil
}

func parseSubscriptionRequest(ctx context.Context, request *http.Request) (subscriptionRequest, error) {
	log := ctx.Value("logger").(logging.Logger)

	result := subscriptionRequest{
		sid:       request.Header.Get("SID"),
		userAgent: request.Header.Get("USER-AGENT"),
		callback:  nil,
		nt:        request.Header.Get("NT"),
		timeout:   -1,
	}

	if len(request.Header.Get("STATEVAR")) > 0 {
		result.statevar = strings.Split(request.Header.Get("STATEVAR"), ",")
	} else {
		result.statevar = []string{}
	}

	callback := request.Header.Get("CALLBACK")
	if len(callback) > 0 {
		callback = strings.TrimPrefix(callback, "<")
		callback = strings.TrimSuffix(callback, ">")
		callbackUrl, err := url.Parse(callback)
		if err != nil {
			log.Warn("[gena] Error while parsing callback url: " + err.Error())
			return subscriptionRequest{}, errors.New("Callback paring error")
		}
		result.callback = callbackUrl
	}

	timeout := request.Header.Get("TIMEOUT")
	if len(timeout) > 0 {
		timeout = strings.TrimPrefix(timeout, "Second-")
		timeoutInt, err := strconv.Atoi(timeout)
		if err != nil {
			log.Warn("[gena] Error while parsing timeout: " + err.Error())
			return subscriptionRequest{}, err
		}
		result.timeout = timeoutInt
	}

	return result, nil
}

func createNewSubscription(subscriptionRequest subscriptionRequest, service Service) (string, error) {
	subscriber := subscriber{
		eventKey:             0,
		httpSuppertedVersion: "1.0",
		userAgent:            subscriptionRequest.userAgent,
	}
	now := time.Now() //TODO "now" is the current implementation of sid (check 1.1.4)
	subscription := subscription{
		sid:        now.UnixNano(),
		subscriber: subscriber,
		service:    service,
		stateVar:   subscriptionRequest.statevar,
		creation:   now,
		timeout:    subscriptionRequest.timeout,
		callback:   subscriptionRequest.callback,
	}

	insertUpdateSubscription <- subscription //TODO Check if it is successful

	return fmt.Sprintf("%d", now.UnixNano()), nil
}

func GenaUnsubscriptionHandler(ctx context.Context, service Service, request *http.Request, response http.ResponseWriter) error {
	log := ctx.Value("logger").(logging.Logger)

	unsubscribeRequestSid := request.Header.Get("SID")

	sid, err := strconv.ParseInt(unsubscribeRequestSid, 10, 64)
	if err != nil {
		generateNegativeResponse(412, response)
		return err
	}

	log.Debug("[gena] Received unsubscribe message, sid: " + unsubscribeRequestSid)

	deleteSubscription <- sid

	response.WriteHeader(200)

	return nil
}

func generatePositiveResponse(subscriptionRequest subscriptionRequest, sid string, response http.ResponseWriter) {
	response.Header().Set("DATE", time.Now().Format(time.RFC1123))
	response.Header().Set("SERVER", ServerUserAgent)
	response.Header().Set("SID", sid)
	response.Header().Set("CONTENT-LENGTH", "0")
	response.Header().Set("TIMEOUT", strconv.Itoa(subscriptionRequest.timeout))
	if len(subscriptionRequest.statevar) > 0 {
		response.Header().Set("ACCEPTED-STATEVAR", utils.StringToCSV(subscriptionRequest.statevar))
	}
}

func generateNegativeResponse(errorCode int, response http.ResponseWriter) {
	response.WriteHeader(errorCode)
}

func GenaSubscriptionDaemon(ctx context.Context) {
	go func() {
		log := ctx.Value("logger").(logging.Logger)

		dbLock := make(chan bool, 1)
		dbLock <- true

		log.Info("[gena] Starting subscription daemon")
		for {
			select {
			case sid := <-deleteSubscription:
				go func() {
					<-dbLock

					service := subscriptionsDB[sid].service
					subscription := subscriptionsDB[sid]
					serviceSubscriptionDB[service.ServiceId] = utils.DeleteElement(serviceSubscriptionDB[service.ServiceId], subscription)
					delete(subscriptionsDB, sid)

					dbLock <- true
				}()
			case subscription := <-insertUpdateSubscription:
				go func() {
					<-dbLock

					subscriptionsDB[subscription.sid] = subscription
					serviceSubscriptionDB[subscription.service.ServiceId] = append(serviceSubscriptionDB[subscription.service.ServiceId], subscription)

					dbLock <- true
				}()
			case notification := <-notificationStateChange:
				go func() {
					<-dbLock

					subscribersOriginal := serviceSubscriptionDB[notification.service.ServiceId]
					subscribers := make([]subscription, len(subscribersOriginal))
					subscribers = slices.Clone(subscribersOriginal)

					dbLock <- true

					sendNotificationToSubscribers(ctx, notification, subscribers)
				}()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func GenaNotifySubscribers(service Service, arguments []device.Argument) {
	stateVariableValues := []stateVariableValue{}

	for _, argument := range arguments {
		for _, stateVariable := range service.SCPD.ServiceStateTable {
			if argument.Name == stateVariable.Name {
				stateVariableValues = append(stateVariableValues, stateVariableValue{
					stateVar: stateVariable,
					value:    argument.Value,
				})
			}
		}
	}

	notificationStateChange <- notification{
		service:             service,
		stateVariableValues: stateVariableValues,
	}
}

func sendNotificationToSubscribers(ctx context.Context, notification notification, subscriptions []subscription) {
	log := ctx.Value("logger").(logging.Logger)

	sendNotification := func(packet TCPPacket) {
		addr, err := net.ResolveTCPAddr("tcp", packet.receiver.IP.String()+":"+strconv.Itoa(packet.receiver.Port))
		if err != nil {
			log.Error("[gena] Error while resolving address: " + err.Error())
			return
		}

		conn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			log.Error("[gena] Error while dial TCP address")
			return
		}
		defer conn.Close()

		_, err = conn.Write([]byte(packet.message))
		if err != nil {
			log.Error("[gena] Error while sending TCP packet")
			return
		}

		messageBuffer := make([]byte, 1024)
		n, err := conn.Read(messageBuffer)
		if err != nil {
			log.Error("[gena] Error while receiving UDP packet")
			return
		}

		log.Info("[gena] Subscription delivery received response: " + string(messageBuffer[:n]))
	}

	for _, subscription := range subscriptions {
		sid := fmt.Sprintf("%d", subscription.sid)
		//usn := ""

		variableValueMap := map[string]string{}
		variableValueMapMulticast := map[string]string{}
		for _, stateVariableValue := range notification.stateVariableValues {
			if stateVariableValue.stateVar.SendEvents && (len(subscription.stateVar) == 0 || slices.Contains(subscription.stateVar, stateVariableValue.stateVar.Name)) {
				variableValueMap[stateVariableValue.stateVar.Name] = stateVariableValue.value
				subscription.subscriber.eventKey++

			}
			if stateVariableValue.stateVar.Multicast {
				variableValueMapMulticast[stateVariableValue.stateVar.Name] = stateVariableValue.value

			}
		}

		go sendNotification(generateNotifyMessage(subscription.callback, sid, subscription.subscriber.eventKey, variableValueMap))
		//go sendNotification(generateNotifyMulticastMessage(subscription.callback, usn, subscription.service.ServiceId, subscription.subscriber.eventKey, Info, variableValueMap))
	}
}

func generateNotifyMessage(host *url.URL, sid string, sequenceNumber int, variableValueMap map[string]string) TCPPacket {
	var result strings.Builder

	// See 4.3.2
	result.WriteString("NOTIFY " + host.Path + " HTTP/1.0\r\n")
	result.WriteString("HOST: delivery " + host.Host + "\r\n")
	result.WriteString("CONTENT-TYPE: text/xml; charset=\"utf-8\"\r\n")
	result.WriteString("NT: upnp:event\r\n")
	result.WriteString("NTS: upnp:propchange\r\n")
	result.WriteString("SID: " + sid + "\r\n")
	result.WriteString("SEQ: " + strconv.Itoa(sequenceNumber) + "\r\n")
	//result.WriteString("CONTENT-LENGTH: 0\r\n")
	result.WriteString("\r\n")

	result.WriteString("<?xml version=\"1.0\"?>\r\n")
	result.WriteString("<e:propertyset xmlns:e=\"urn:schemas-upnp-org:event-1-0\">\r\n")
	result.WriteString("<e:property>\r\n")

	for key, value := range variableValueMap {
		result.WriteString("<" + key + ">" + value + "</" + key + ">\r\n")
	}

	result.WriteString("</e:property>\r\n")
	result.WriteString("</e:propertyset>\r\n")

	receiver, _ := net.ResolveTCPAddr("tcp", host.Host)
	return TCPPacket{
		message:  result.String(),
		receiver: *receiver,
	}
}

func generateNotifyMulticastMessage(host *url.URL, usn string, serviceId string, sequenceNumber int, genaMulticastEventLevels GenaMulticastEventLevels, variableValueMap map[string]string) UDPPacket {
	var result strings.Builder

	// See 4.3.3
	result.WriteString("NOTIFY * HTTP/1.0\r\n")
	result.WriteString("HOST: " + genaMulticastNotificationAddress + ":" + strconv.Itoa(genaMulticastNotificationPort) + "\r\n")
	result.WriteString("CONTENT-TYPE: text/xml; charset=\"utf-8\"\r\n")
	result.WriteString("USN: " + usn + "\r\n")
	result.WriteString("SVCID: " + serviceId + "\r\n")
	result.WriteString("NT: upnp:event\r\n")
	result.WriteString("NTS: upnp:propchange\r\n")
	result.WriteString("SEQ: " + strconv.Itoa(sequenceNumber) + "\r\n")
	result.WriteString("LVL: " + genaMulticastEventLevels.String() + "\r\n")
	//result.WriteString("CONTENT-LENGTH: 0\r\n")
	result.WriteString("\r\n")

	result.WriteString("<?xml version=\"1.0\"?>\r\n")
	result.WriteString("<e:propertyset xmlns:e=\"urn:schemas-upnp-org:event-1-0\">\r\n")
	result.WriteString("<e:property>\r\n")

	for key, value := range variableValueMap {
		result.WriteString("<" + key + ">" + value + "</" + key + ">\r\n")
	}

	result.WriteString("</e:property>\r\n")
	result.WriteString("</e:propertyset>\r\n")

	receiver, _ := net.ResolveUDPAddr("udp", host.Host)
	return UDPPacket{
		message:  result.String(),
		receiver: *receiver,
	}
}
