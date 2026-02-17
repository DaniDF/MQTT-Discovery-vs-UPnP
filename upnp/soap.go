package upnp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	device "mobile.dani.df/device-service"
	"mobile.dani.df/logging"
)

type Envelope struct {
	XMLName       xml.Name `xml:"Envelope"`
	Xmlns         string   `xml:"xmlns,attr"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	Body          EnvelopeBody
}

type EnvelopeBody struct {
	XMLName    xml.Name   `xml:"Body"`
	ActionName ActionName `xml:",any"`
}

type ActionName struct {
	XMLName       xml.Name
	ArgumentNames []ArgumentName `xml:",any"`
}

type ArgumentName struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func SoapControlHandler(ctx context.Context, rootDevice RootDevice, dev device.Device, request *http.Request, response http.ResponseWriter) error {
	log := ctx.Value("logger").(logging.Logger)
	/*
		HTTP/1.0 200 OK
		CONTENT-TYPE: text/xml; charset="utf-8"
		DATE: when response was generated
		SERVER: OS/version UPnP/2.0 product/version
		CONTENT-LENGTH: bytes in body <?xml version="1.0"?>

		<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
			<s:Body>
				<u:actionNameResponse xmlns:u="urn:schemas-upnp-org:service:serviceType:v">
					<argumentName>out arg value</argumentName>
					<!-- other out args and their values go here, if any -->
				</u:actionNameResponse>
			</s:Body>
		</s:Envelope>
	*/

	data, err := io.ReadAll(request.Body)
	if err != nil {
		log.Error("[soap] Error reading request body: " + err.Error())
		return err
	}

	envelope := Envelope{}
	err = xml.Unmarshal(data, &envelope)
	if err != nil {
		log.Error("[soap] Error unmarshaling envelope: " + err.Error())
		return err
	}

	log.Info("[soap] RPC Requested: " + envelope.Body.ActionName.XMLName.Space)

	dev.ControlFunc(device.Argument{
		Name:  envelope.Body.ActionName.ArgumentNames[0].XMLName.Space,
		Value: envelope.Body.ActionName.ArgumentNames[0].Value,
	})

	fmt.Fprint(response, "<s:Envelope xmlns:s=\"http://schemas.xmlsoap.org/soap/envelope/\" s:encodingStyle=\"http://schemas.xmlsoap.org/soap/encoding/\">\n")
	fmt.Fprint(response, "<s:Body>\n")
	fmt.Fprint(response, "<u:actionNameResponse xmlns:u=\"urn:schemas-upnp-org:service:serviceType:v\">\n")

	fmt.Fprint(response, "</u:actionNameResponse>\n")
	fmt.Fprint(response, "</s:Body>\n")
	fmt.Fprint(response, "</s:Envelope>\n")

	return nil
}
