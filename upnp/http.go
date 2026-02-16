package upnp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"mobile.dani.df/logging"
)

const (
	httpServerPort = 8080
)

type UDPPacket struct {
	source   net.UDPAddr
	receiver net.UDPAddr
	message  string
}

func (m UDPPacket) String() string {
	return m.source.String() + " says " + m.message
}

var httpServerAddress = GetLocalIP()

func HttpServer(ctx context.Context, rootDevice RootDevice, devicePresentationUrl string) {
	go func() {
		log := ctx.Value("logger").(logging.Logger)

		http.HandleFunc(devicePresentationUrl, func(resp http.ResponseWriter, req *http.Request) {
			deviceDescriptionHandler(ctx, rootDevice, req, resp)
		})
		for _, embeddedDevice := range rootDevice.Device.EmbeddedDevices {
			http.HandleFunc(embeddedDevice.PresentationURL, func(resp http.ResponseWriter, req *http.Request) {
				deviceDescriptionHandler(ctx, rootDevice, req, resp)
			})
		}

		for _, service := range rootDevice.Device.ServiceList {
			http.HandleFunc(service.SCPDURL, func(resp http.ResponseWriter, req *http.Request) { scpdURLHandler(ctx, rootDevice, req, resp) })
			http.HandleFunc(service.ControlURL, func(resp http.ResponseWriter, req *http.Request) { serviceControlHandler(ctx, rootDevice, req, resp) })
		}
		for _, embeddedDevice := range rootDevice.Device.EmbeddedDevices {
			for _, service := range embeddedDevice.ServiceList {
				http.HandleFunc(service.SCPDURL, func(resp http.ResponseWriter, req *http.Request) { scpdURLHandler(ctx, rootDevice, req, resp) })
				http.HandleFunc(service.ControlURL, func(resp http.ResponseWriter, req *http.Request) { serviceControlHandler(ctx, rootDevice, req, resp) })
			}
		}

		log.Info("[http] Listening for request")
		err := http.ListenAndServe(httpServerAddress+":"+strconv.Itoa(httpServerPort), nil)
		log.Error("[http] Error occurred while listen and serve: " + err.Error())
	}()
}

func deviceDescriptionHandler(ctx context.Context, rootDevice RootDevice, request *http.Request, response http.ResponseWriter) {
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
		fmt.Fprint(response, service.ControlHandler())
	}

	serviceNotFoundHandler := func() {
		response.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(response, "Sorry come later I'll Harder, Better, Faster, Stronger")
	}

	serviceRequestHandler(ctx, rootDevice, request, response, func(s Service) string {
		return s.ControlURL
	}, serviceFoundHandler, serviceNotFoundHandler)
}

func serviceRequestHandler(ctx context.Context, rootDevice RootDevice, request *http.Request, response http.ResponseWriter, extractor func(Service) string, serviceFoundHandler func(Service), serviceNotFoundHandler func()) {
	log := ctx.Value("logger").(logging.Logger)

	log.Info("[http] Request from " + request.RemoteAddr + " resource " + request.RequestURI)

	prepareOKresponse := func() {
		log.Info("[http] Request from " + request.RemoteAddr + " resource " + request.RequestURI + " -> OK - FOUND")
		response.Header().Set("Content-Type", "application/xml")
		response.WriteHeader(http.StatusOK)
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
