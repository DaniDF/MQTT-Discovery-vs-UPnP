package upnp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	device "mobile.dani.df/device-service"
	"mobile.dani.df/logging"
	"mobile.dani.df/utils"
)

const soapTimeoutSeconds = 30 // Timeout for a SOAP request (see 3.2.5)

type Envelope struct {
	XMLName       xml.Name `xml:"Envelope"`
	Xmlns         string   `xml:"xmlns,attr"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	Body          EnvelopeBody
}

type EnvelopeBody struct {
	XMLName    xml.Name         `xml:"Body"`
	ActionName ActualActionName `xml:",any"`
}

type ActualActionName struct {
	XMLName       xml.Name
	ArgumentNames []ActualArgumentName `xml:",any"`
}

type ActualArgumentName struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

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

type ActionNameResponse struct {
	XMLName       xml.Name
	Xmlns         string               `xml:"xmlns,attr"`
	ArgumentNames []ActualArgumentName `xml:",any"`
}

/*
	HTTP/1.0 500 Internal Server Error
	CONTENT-TYPE: text/xml; charset="utf-8"
	DATE: when response was generated
	SERVER: OS/version UPnP/2.0 product/version
	CONTENT-LENGTH: bytes in body <?xml version="1.0"?>

	<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
		<s:Body>
			<s:Fault>
				<faultcode>s:Client</faultcode>
				<faultstring>UPnPError</faultstring>
				<detail>
					<UPnPError xmlns="urn:schemas-upnp-org:control-1-0">
						<errorCode>error code</errorCode>
						<errorDescription>error string</errorDescription>
					</UPnPError>
				</detail>
			</s:Fault>
		</s:Body>
	</s:Envelope>
*/

// Handle a SOAP request
func SoapControlHandler(ctx context.Context, deviceService Service, request *http.Request, response http.ResponseWriter) error {
	log := ctx.Value("logger").(logging.Logger)

	data, err := io.ReadAll(request.Body)
	if err != nil {
		log.Error("[soap] Error reading request body: " + err.Error())
		generateErrorResponse(501, "Error while reading request body", response)
		return err
	}

	envelope := Envelope{}
	err = xml.Unmarshal(data, &envelope)
	if err != nil {
		log.Error("[soap] Error unmarshaling envelope: " + err.Error())
		generateErrorResponse(501, "Error unmarshaling envelope", response)
		return err
	}

	log.Info("[soap] RPC Requested: " + envelope.Body.ActionName.XMLName.Space)

	log.Debug("[soap] Action name: " + envelope.Body.ActionName.XMLName.Local)

	formalAction, findAction := utils.FindFirst(deviceService.SCPD.actionList, func(action FormalAction) bool {
		return action.Name == envelope.Body.ActionName.XMLName.Local
	})
	if !findAction {
		generateErrorResponse(401, "Action requested not implemented by this service", response)
		return errors.New("Action not found")
	}

	formalInArguments := utils.Find(formalAction.ArgumentList, func(argument FormalArgument) bool {
		return argument.Direction == In
	})

	inArguments, err := getInArguments(formalInArguments, envelope.Body.ActionName.ArgumentNames)
	if err != nil {
		generateErrorResponse(402, "Actual arguments do not match formal argument", response)
		log.Error("[soap] Error while assigning formal-arguments to actual-arguments")
		return err
	}

	deviceResponseChan := make(chan device.Response)
	go func() {
		deviceResponseChan <- deviceService.Handler(inArguments...)
	}()

	var deviceResponse device.Response
	select {
	case deviceResponse = <-deviceResponseChan:
		if deviceResponse.ErrorCode != 0 {
			generateErrorResponse(501, "Execution failed: "+deviceResponse.ErrorMessage, response)
			log.Warn("[soap] Device execution failed. Code: " + strconv.Itoa(deviceResponse.ErrorCode) + " - " + deviceResponse.ErrorMessage)
			return nil
		}
	case <-time.After(soapTimeoutSeconds * time.Second):
		generateErrorResponse(501, "Timeout", response)
		return errors.New("Timeout")
	}

	formalOutArgument, findOutArgument := utils.FindFirst(formalAction.ArgumentList, func(argument FormalArgument) bool {
		return argument.Direction == Out
	})

	var resultArgument ActualArgumentName
	if findOutArgument {
		resultArgument = ActualArgumentName{
			XMLName: xml.Name{
				Local: formalOutArgument.Name,
			},
			Value: deviceResponse.Value,
		}
	}

	result, err := xml.Marshal(ActionNameResponse{
		/* Should be in this way but actual implementation does not support it
		XMLName: xml.Name{
			Local: envelope.Body.ActionName.XMLName.Space + "Response",
		},*/
		Xmlns:         envelope.Body.ActionName.XMLName.Space,
		ArgumentNames: []ActualArgumentName{resultArgument},
	})
	if err != nil {
		log.Error("[soap] Error while mashaling the response")
		return err
	}
	generetePositiveResponse(response, string(result))

	return nil
}

// Checks if all the requested formalArgument are present.
// In case of more actualArgument than needed, the surplus is discarded.
func getInArguments(formalArguments []FormalArgument, actualArguments []ActualArgumentName) ([]device.Argument, error) {
	result := []device.Argument{}

	for _, formalArgument := range formalArguments {
		actualArgument, findActualArgument := utils.FindFirst(actualArguments, func(actual ActualArgumentName) bool {
			return actual.XMLName.Local == formalArgument.Name
		})
		if !findActualArgument {
			return []device.Argument{}, errors.New("Argument not found")
		}

		result = append(result, device.Argument{
			Name:  actualArgument.XMLName.Local,
			Value: actualArgument.Value,
		})
	}

	return result, nil
}

// Generates a positive response
func generetePositiveResponse(response http.ResponseWriter, actionNameResponseString string) {
	response.WriteHeader(http.StatusOK)
	fmt.Fprint(response, "<s:Envelope xmlns:s=\"http://schemas.xmlsoap.org/soap/envelope/\" s:encodingStyle=\"http://schemas.xmlsoap.org/soap/encoding/\">\n")
	fmt.Fprint(response, "<s:Body>\n")

	fmt.Fprint(response, actionNameResponseString)

	fmt.Fprint(response, "</s:Body>\n")
	fmt.Fprint(response, "</s:Envelope>\n")
}

// Generates a negative response
func generateErrorResponse(errorCode int, errorMessage string, response http.ResponseWriter) {
	response.Header().Set("CONTENT-TYPE", "text/xml; charset=\"utf-8\"")
	response.Header().Set("DATE", time.Now().Format(time.RFC1123))
	response.Header().Set("SERVER", ServerUserAgent)
	response.WriteHeader(http.StatusInternalServerError)

	fmt.Fprint(response, "<s:Envelope xmlns:s=\"http://schemas.xmlsoap.org/soap/envelope/\" s:encodingStyle=\"http://schemas.xmlsoap.org/soap/encoding/\">\n")
	fmt.Fprint(response, "<s:Body>\n")
	fmt.Fprint(response, "<s:Fault>\n")
	fmt.Fprint(response, "<faultcode>s:Client</faultcode>\n")
	fmt.Fprint(response, "<faultstring>UPnPError</faultstring>\n")
	fmt.Fprint(response, "<detail>\n")
	fmt.Fprint(response, "<UPnPError xmlns=\"urn:schemas-upnp-org:control-1-0\">\n")
	fmt.Fprint(response, "<errorCode>"+strconv.Itoa(errorCode)+"</errorCode>\n")
	fmt.Fprint(response, "<errorDescription>"+errorMessage+"</errorDescription>\n")
	fmt.Fprint(response, "</UPnPError>\n")
	fmt.Fprint(response, "</detail>\n")
	fmt.Fprint(response, "</s:Fault>\n")
	fmt.Fprint(response, "</s:Body>\n")
	fmt.Fprint(response, "</s:Envelope>\n")
}
