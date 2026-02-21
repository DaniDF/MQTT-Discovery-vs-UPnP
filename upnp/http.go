package upnp

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"mobile.dani.df/logging"
	"mobile.dani.df/utils"
)

type UDPPacket struct {
	source   net.UDPAddr
	receiver net.UDPAddr
	message  string
}

func (m UDPPacket) String() string {
	return m.source.String() + " says " + m.message
}

// --------------------------------------------------------------------------------------
// For upnp control point
// --------------------------------------------------------------------------------------

func RetrieveDeviceDescriptor(ctx context.Context, maybeDevice MSearchResult) (string, error) {
	log := ctx.Value("logger").(logging.Logger)

	responseHttp, err := http.Get(maybeDevice.Location)
	if err != nil {
		log.Error("Error while getting the device locator from: " + maybeDevice.Location)
	}
	defer responseHttp.Body.Close()

	response, err := io.ReadAll(responseHttp.Body)

	return string(response), err
}

// Sends the specified request
func SendRequest(request *http.Request) (*http.Response, error) {
	httpClient := http.Client{}
	return httpClient.Do(request)
}

// --------------------------------------------------------------------------------------
// For upnp device
// --------------------------------------------------------------------------------------

var httpServerAddress = utils.GetLocalIP()

type HttpServer struct {
	ctx      context.Context
	listener net.Listener
	Port     int
}

func NewHttpServer(ctx context.Context) (HttpServer, error) {
	log := ctx.Value("logger").(logging.Logger)

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Error("[http] Error while starting listening: " + err.Error())
		return HttpServer{}, nil
	}

	return HttpServer{
		ctx:      ctx,
		listener: listener,
		Port:     listener.Addr().(*net.TCPAddr).Port,
	}, nil
}

func (httpServer HttpServer) ServeRootDevice(rootDevice RootDevice, devicePresentationUrl string) {
	go func() {
		log := httpServer.ctx.Value("logger").(logging.Logger)

		httpMux := http.NewServeMux()

		httpMux.HandleFunc(devicePresentationUrl, func(resp http.ResponseWriter, req *http.Request) {
			deviceDescriptionHandler(httpServer.ctx, rootDevice, httpServer.Port, req, resp)
		})
		for _, embeddedDevice := range rootDevice.Device.EmbeddedDevices {
			httpMux.HandleFunc(embeddedDevice.PresentationURL, func(resp http.ResponseWriter, req *http.Request) {
				deviceDescriptionHandler(httpServer.ctx, rootDevice, httpServer.Port, req, resp)
			})
		}

		for _, service := range rootDevice.Device.ServiceList {
			httpMux.HandleFunc(service.SCPDURL, func(resp http.ResponseWriter, req *http.Request) {
				scpdURLHandler(httpServer.ctx, rootDevice, req, resp)
			})
			httpMux.HandleFunc(service.ControlURL, func(resp http.ResponseWriter, req *http.Request) {
				serviceControlHandler(httpServer.ctx, rootDevice, req, resp)
			})
			httpMux.HandleFunc(service.EventSubURL, func(resp http.ResponseWriter, req *http.Request) {
				serviceEventHandler(httpServer.ctx, rootDevice, req, resp)
			})
		}
		for _, embeddedDevice := range rootDevice.Device.EmbeddedDevices {
			for _, service := range embeddedDevice.ServiceList {
				httpMux.HandleFunc(service.SCPDURL, func(resp http.ResponseWriter, req *http.Request) {
					scpdURLHandler(httpServer.ctx, rootDevice, req, resp)
				})
				httpMux.HandleFunc(service.ControlURL, func(resp http.ResponseWriter, req *http.Request) {
					serviceControlHandler(httpServer.ctx, rootDevice, req, resp)
				})
			}
		}

		log.Info("[http] Listening for request at " + httpServer.listener.Addr().String())
		err := http.Serve(httpServer.listener, httpMux)
		log.Error("[http] Error occurred while listen and serve: " + err.Error())
	}()
}

func deviceDescriptionHandler(ctx context.Context, rootDevice RootDevice, httpServerPort int, request *http.Request, response http.ResponseWriter) {
	log := ctx.Value("logger").(logging.Logger)

	foundDeviceHandler := func(device SerializableXML) {
		log.Info("[http] Request from " + request.RemoteAddr + " resource " + request.RequestURI + " -> OK - FOUND")
		response.Header().Set("Content-Type", "application/xml")
		response.WriteHeader(http.StatusOK)
		fmt.Fprint(response, device.StringXML())
	}

	flagFoundDevice := false
	if rootDevice.Device.PresentationURL == "http://"+httpServerAddress+":"+strconv.Itoa(httpServerPort)+request.RequestURI {
		flagFoundDevice = true
		foundDeviceHandler(rootDevice)
	}
	if !flagFoundDevice {
		for _, embeddedDevice := range rootDevice.Device.EmbeddedDevices {
			if embeddedDevice.PresentationURL == "http://"+httpServerAddress+":"+strconv.Itoa(httpServerPort)+request.RequestURI {
				flagFoundDevice = true
				foundDeviceHandler(embeddedDevice)
			}
		}
	}

	if !flagFoundDevice {
		log.Warn("[http] Request from " + request.RemoteAddr + " resource " + request.RequestURI + " -> 404 - NOT FOUND")
		response.Header().Set("Content-Type", "text/plain")
		response.WriteHeader(http.StatusNotFound)
		fmt.Fprint(response, "Sorry you requested some device descriptor that is not here. Are you sure you are talking with the right device?")
	}
}

func scpdURLHandler(ctx context.Context, rootDevice RootDevice, request *http.Request, response http.ResponseWriter) {
	serviceFoundHandler := func(service Service) {
		response.WriteHeader(http.StatusOK)
		fmt.Fprint(response, service.SCPD.StringXML())
	}

	serviceNotFoundHandler := func() {
		response.WriteHeader(http.StatusNotFound)
		fmt.Fprint(response, "Sorry you requested some service descriptor that is not advertised. Are you sure you are talking with the right device?")
	}

	serviceRequestHandler(ctx, rootDevice, request, response, func(s Service) string {
		return s.SCPDURL
	}, serviceFoundHandler, serviceNotFoundHandler)
}

func serviceControlHandler(ctx context.Context, rootDevice RootDevice, request *http.Request, response http.ResponseWriter) {
	serviceFoundHandler := func(service Service) {
		SoapControlHandler(ctx, service, request, response)
	}

	serviceNotFoundHandler := func() {
		response.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(response, "Sorry come later I'll Harder, Better, Faster, Stronger")
	}

	serviceRequestHandler(ctx, rootDevice, request, response, func(s Service) string {
		return s.ControlURL
	}, serviceFoundHandler, serviceNotFoundHandler)
}

func serviceEventHandler(ctx context.Context, rootDevice RootDevice, request *http.Request, response http.ResponseWriter) {
	serviceFoundHandler := func(service Service) {
		switch request.Method {
		case "SUBSCRIBE":
			GenaSubscriptionHandler(ctx, service, request, response)
		case "UNSUBSCRIBE":
			GenaUnsubscriptionHandler(ctx, service, request, response)
		}
	}

	serviceNotFoundHandler := func() {
		response.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(response, "Sorry come later I'll Harder, Better, Faster, Stronger")
	}

	serviceRequestHandler(ctx, rootDevice, request, response, func(s Service) string {
		return s.EventSubURL
	}, serviceFoundHandler, serviceNotFoundHandler)
}

func serviceRequestHandler(ctx context.Context, rootDevice RootDevice, request *http.Request, response http.ResponseWriter, extractor func(Service) string, serviceFoundHandler func(Service), serviceNotFoundHandler func()) {
	log := ctx.Value("logger").(logging.Logger)

	log.Info("[http] Request from " + request.RemoteAddr + " resource " + request.RequestURI)

	prepareOKresponse := func() {
		log.Info("[http] Request from " + request.RemoteAddr + " resource " + request.RequestURI + " -> OK - FOUND")
		response.Header().Set("Content-Type", "text/xml")
	}

	prepareBadresponse := func() {
		log.Warn("[http] Request from " + request.RemoteAddr + " resource " + request.RequestURI + " -> 404 - NOT FOUND")
		response.Header().Set("Content-Type", "text/plain")
	}

	flagFoundService := false
	for _, service := range rootDevice.Device.ServiceList {
		if extractor(service) == request.RequestURI {

			flagFoundService = true
			prepareOKresponse()
			serviceFoundHandler(service)
		}
	}
	if !flagFoundService {
		for _, embeddedDevice := range rootDevice.Device.EmbeddedDevices {
			for _, service := range embeddedDevice.ServiceList {
				if extractor(service) == request.RequestURI {
					flagFoundService = true
					prepareOKresponse()
					serviceFoundHandler(service)
				}
			}
		}
	}

	if !flagFoundService {
		prepareBadresponse()
		serviceNotFoundHandler()
	}
}
