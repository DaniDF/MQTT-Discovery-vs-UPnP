package upnp

import (
	"errors"
	"strconv"
	"strings"
)

/* See 2.3
<root xmlns="urn:schemas-upnp-org:device-1-0" configId="1">
	<specVersion>
		<major>1</major>
		<minor>0</minor>
	</specVersion>
	<device>
		<deviceType>{{DEV_TYPE}}</deviceType>
		<UDN>uuid:55076f6e-6b79-4d65-6401-00d0b811d10b</UDN>
		<friendlyName>MediaServer</friendlyName>
		<manufacturer>Manufacturer, Inc.</manufacturer>
		<manufacturerURL>http://www.manufacturer.com</manufacturerURL>
		<modelName>MediaServer 1.0</modelName>
		<modelURL>http://www.manufacturer.com/MediaServer</modelURL>
		<modelDescription>Media Server</modelDescription>
		<modelNumber>Media Server Home</modelNumber>
		<serialNumber>123345-50023</serialNumber>
		<UPC>1009234789893</UPC>
		<presentationURL>http://{{IP}}:{{PORT}}/</presentationURL>
		<iconList>
			<icon>
				<mimetype>image/jpeg</mimetype>
				<height>48</height>
				<width>48</width>
				<depth>24</depth>
				<url>/images/icon-48x48.jpg</url>
			</icon>
			<icon>
				<mimetype>image/jpeg</mimetype>
				<height>120</height>
				<width>120</width>
				<depth>24</depth>
				<url>/images/icon-120x120.jpg</url>
			</icon>
		</iconList>
		<serviceList>
			<service>
				<serviceType>urn:schemas-upnp-org:service:ConnectionManager:1</serviceType>
				<serviceId>urn:upnp-org:serviceId:ConnectionManager</serviceId>
				<SCPDURL>/ConnectionManager.xml</SCPDURL>
				<eventSubURL>/ConnectionManager/Event</eventSubURL>
				<controlURL>/ConnectionManager/Control</controlURL>
			</service>
			<service>
				<serviceType>urn:schemas-upnp-org:service:ContentDirectory:1</serviceType>
				<serviceId>urn:upnp-org:serviceId:ContentDirectory</serviceId>
				<SCPDURL>/ContentDirectory.xml</SCPDURL>
				<eventSubURL>/ContentDirectory/Event</eventSubURL>
				<controlURL>/ContentDirectory/Control</controlURL>
			</service>
		</serviceList>
		<deviceList>
		<deviceList>

		</deviceList>
	</device>
</root>
*/

/* See 2.5
<?xml version="1.0"?>
<scpd xmlns="urn:schemas-upnp-org:service-1-0" xmlns:dt1="urn:domain-name:more-datatypes" xmlns:dt2="urn:domain-name:vendor-datatypes" configId="configuration number">
	<specVersion>
		<major>2</major>
		<minor>0</minor>
	</specVersion>
	<actionList>
		<action>
			<name>actionName</name>
			<argumentList>
				<argument>
					<name>argumentNameIn1</name>
					<direction>in</direction>
					<relatedStateVariable>stateVariableName</relatedStateVariable>
				</argument>
				<argument>
					<name>argumentNameOut1</name>
					<direction>out</direction>
					<retval/>
					<relatedStateVariable>stateVariableName</relatedStateVariable>
				</argument>
				<argument>
					<name>argumentNameOut2</name>
					<direction>out</direction>
					<relatedStateVariable>stateVariableName</relatedStateVariable>
				</argument>
			</argumentList>
		</action>
	</actionList>
	<serviceStateTable>
		<stateVariable sendEvents="yes"|"no" multicast="yes"|"no">
			<name>variableName</name>
			<dataType>basic data type</dataType>
			<defaultValue>default value</defaultValue>
			<allowedValueRange>
				<minimum>minimum value</minimum>
				<maximum>maximum value</maximum>
				<step>increment value</step>
			</allowedValueRange>
		</stateVariable>
		<stateVariable sendEvents="yes"|"no" multicast="yes"|"no">
			<name>variableName</name>
			<dataType type="dt1:variable data type">string</dataType>
			<defaultValue>default value</defaultValue>
			<allowedValueList>
				<allowedValue>enumerated value</allowedValue>
			</allowedValueList>
		</stateVariable>
		<stateVariable sendEvents="yes"|"no" multicast="yes"|"no">
			<name>variableName</name>
			<dataType type="dt2:vendor data type">string</dataType>
			<defaultValue>default value</defaultValue>
		</stateVariable>
	</serviceStateTable>
</scpd>
*/

type RootDevice struct {
	SpecVersion SpecVersion
	Device      Device
}

type SpecVersion struct {
	Major string
	Minor string
}

type Device struct {
	DeviceType       string
	UDN              string
	FriendlyName     string
	Manufacturer     string
	ManufacturerURL  string
	ModelName        string
	ModelURL         string
	ModelDescription string
	ModelNumber      string
	SerialNumber     string
	UPC              string
	PresentationURL  string
	IconList         []Icon
	ServiceList      []Service
	EmbeddedDevices  []Device
}

type Icon struct {
	Mimetype string
	Height   string
	Width    string
	Depth    string
	Url      string
}

type Service struct {
	ServiceType    string
	ServiceId      string
	SCPDURL        string
	EventSubURL    string
	ControlURL     string
	ControlHandler func() string

	SCPD Spcd
}

type Spcd struct {
	SpecVersion       SpecVersion
	actionList        []Action
	ServiceStateTable []*StateVariable
}

// Add the provided action to the SPCD.
// Rises an "StateVariable not found" if at least one of the arguments has a RelatedStateVariable not present in ServiceStateTable.
func (spcd *Spcd) AddAction(action Action) error {
	flagFoundArgumentStateVariable := true
	for _, argument := range action.ArgumentList {

		flagFoundStateVariable := false
		for i := range spcd.ServiceStateTable {
			flagFoundStateVariable = flagFoundStateVariable || spcd.ServiceStateTable[i] == argument.RelatedStateVariable
		}

		flagFoundArgumentStateVariable = flagFoundArgumentStateVariable && flagFoundStateVariable
	}

	if !flagFoundArgumentStateVariable {
		return errors.New("StateVariable not found")
	}

	spcd.actionList = append(spcd.actionList, action)

	return nil
}

type Action struct {
	Name         string
	ArgumentList []Argument
}

type Argument struct {
	Name                 string
	Direction            ArgumentDirection
	RelatedStateVariable *StateVariable
}

type ArgumentDirection string

const (
	In  ArgumentDirection = "in"
	Out ArgumentDirection = "out"
)

func (argumentDirection ArgumentDirection) String() string {
	switch argumentDirection {
	case In:
		return "in"
	case Out:
		return "out"
	default:
		return "in"
	}
}

type StateVariable struct {
	SendEvents        bool
	Multicast         bool
	Name              string
	DataType          string
	DefaultValue      string
	AllowedValueRange *ValueRange
	AllowedValueList  []Value
}

type ValueRange struct {
	Minimum int //TODO All should be comparable
	Maximum int
	Step    int
}

type Value struct {
	AllowedValue int //TODO Should be comparable
}

type SerializableXML interface {
	StringXML() string
}

// Generates a string compatible with the specifications (see 2.3)
func (rootDevice RootDevice) StringXML() string {
	var result strings.Builder

	result.WriteString("<root xmlns=\"urn:schemas-upnp-org:device-1-0\" configId=\"1\">\n")
	result.WriteString(rootDevice.SpecVersion.StringXML())

	result.WriteString(rootDevice.Device.StringXML())

	result.WriteString("</root>\n")

	return result.String()
}

// Generates a string compatible with the specifications (see 2.3)
func (specVersion SpecVersion) StringXML() string {
	var result strings.Builder

	result.WriteString("<specVersion>\n")
	result.WriteString("<major>" + specVersion.Major + "</major>\n")
	result.WriteString("<minor>" + specVersion.Minor + "</minor>\n")
	result.WriteString("</specVersion>\n")

	return result.String()
}

// Generates a string compatible with the specifications (see 2.3)
func (device Device) StringXML() string {
	var result strings.Builder

	result.WriteString("<device>\n")
	result.WriteString("<deviceType>" + device.DeviceType + "</deviceType>\n")
	result.WriteString("<UDN>" + device.UDN + "</UDN>\n")
	result.WriteString("<friendlyName>" + device.FriendlyName + "</friendlyName>\n")
	result.WriteString("<manufacturer>" + device.Manufacturer + "</manufacturer>\n")
	result.WriteString("<manufacturerURL>" + device.ManufacturerURL + "</manufacturerURL>\n")
	result.WriteString("<modelName>" + device.ModelName + "</modelName>\n")
	result.WriteString("<modelURL>" + device.ModelURL + "</modelURL>\n")
	result.WriteString("<modelDescription>" + device.ModelDescription + "</modelDescription>\n")
	result.WriteString("<modelNumber>" + device.ModelNumber + "</modelNumber>\n")
	result.WriteString("<serialNumber>" + device.SerialNumber + "</serialNumber>\n")
	result.WriteString("<UPC>" + device.UPC + "</UPC>\n")
	result.WriteString("<presentationURL>" + device.PresentationURL + "</presentationURL>\n")

	result.WriteString("<iconList>\n")
	for _, icon := range device.IconList {
		result.WriteString(icon.StringXML())
	}
	result.WriteString("</iconList>\n")

	result.WriteString("<serviceList>\n")
	for _, service := range device.ServiceList {
		result.WriteString(service.StringXML())
	}
	result.WriteString("</serviceList>\n")

	if len(device.EmbeddedDevices) > 0 {
		result.WriteString("<deviceList>\n")
		for _, embeddedDevice := range device.EmbeddedDevices {
			result.WriteString(embeddedDevice.StringXML())
		}
		result.WriteString("</deviceList>\n")
	}

	result.WriteString("</device>\n")

	return result.String()
}

// Generates a string compatible with the specifications (see 2.3)
func (icon Icon) StringXML() string {
	var result strings.Builder

	result.WriteString("<icon>\n")
	result.WriteString("<mimetype>" + icon.Mimetype + "</mimetype>\n")
	result.WriteString("<height>" + icon.Height + "</height>\n")
	result.WriteString("<width>" + icon.Width + "</width>\n")
	result.WriteString("<depth>" + icon.Depth + "</depth>\n")
	result.WriteString("<url>" + icon.Url + "</url>\n")
	result.WriteString("</icon>\n")

	return result.String()
}

// Generates a string compatible with the specifications (see 2.3)
func (service Service) StringXML() string {
	var result strings.Builder

	result.WriteString("<service>\n")
	result.WriteString("<serviceType>" + service.ServiceType + "</serviceType>\n")
	result.WriteString("<serviceId>" + service.ServiceId + "</serviceId>\n")
	result.WriteString("<SCPDURL>" + service.SCPDURL + "</SCPDURL>\n")
	result.WriteString("<eventSubURL>" + service.EventSubURL + "</eventSubURL>\n")
	result.WriteString("<controlURL>" + service.ControlURL + "</controlURL>\n")
	result.WriteString("</service>\n")

	return result.String()
}

func (spcd Spcd) StringXML() string {
	var result strings.Builder

	result.WriteString("<?xml version=\"1.0\"?>\n")
	result.WriteString("<scpd xmlns=\"urn:schemas-upnp-org:service-1-0\" xmlns:dt1=\"urn:domain-name:more-datatypes\" xmlns:dt2=\"urn:domain-name:vendor-datatypes\" configId=\"1\">\n") //TODO da capire

	result.WriteString(spcd.SpecVersion.StringXML())

	result.WriteString("<actionList>\n")

	for _, action := range spcd.actionList {
		result.WriteString(action.StringXML())
	}
	result.WriteString("</actionList>\n")

	result.WriteString("<serviceStateTable>\n")
	for _, stateVariable := range spcd.ServiceStateTable {
		result.WriteString(stateVariable.StringXML())
	}
	result.WriteString("</serviceStateTable>\n")

	result.WriteString("</scpd>\n")

	return result.String()
}

func (action Action) StringXML() string {
	var result strings.Builder

	result.WriteString("<action>\n")
	result.WriteString("<name>" + action.Name + "</name>\n")

	result.WriteString("<argumentList>\n")
	for _, argument := range action.ArgumentList {
		result.WriteString(argument.StringXML())
	}
	result.WriteString("</argumentList>\n")

	result.WriteString("</action>\n")

	return result.String()
}

func (argument Argument) StringXML() string {
	var result strings.Builder

	result.WriteString("<argument>\n")
	result.WriteString("<name>" + argument.Name + "</name>\n")
	result.WriteString("<direction>" + argument.Direction.String() + "</direction>\n")
	result.WriteString("<relatedStateVariable>" + argument.RelatedStateVariable.Name + "</relatedStateVariable>\n")
	result.WriteString("</argument>\n")

	return result.String()
}

func (stateVariable StateVariable) StringXML() string {
	var result strings.Builder

	sendEvents := ""
	if stateVariable.SendEvents {
		sendEvents = "yes"
	} else {
		sendEvents = "no"
	}
	multicast := ""
	if stateVariable.Multicast {
		multicast = "yes"
	} else {
		multicast = "no"
	}
	result.WriteString("<stateVariable sendEvents=\"" + sendEvents + "\" multicast=\"" + multicast + "\">\n")
	result.WriteString("<name>" + stateVariable.Name + "</name>\n")
	result.WriteString("<dataType>" + stateVariable.DataType + "</dataType>\n")

	if stateVariable.AllowedValueRange != nil {
		result.WriteString(stateVariable.AllowedValueRange.StringXML())
	}
	if len(stateVariable.AllowedValueList) > 0 {
		result.WriteString("<allowedValueList>\n")
		for _, value := range stateVariable.AllowedValueList {
			result.WriteString(value.StringXML())
		}
		result.WriteString("</allowedValueList>\n")
	}

	result.WriteString("</stateVariable>\n")

	return result.String()
}

func (valueRange ValueRange) StringXML() string {
	var result strings.Builder

	result.WriteString("<allowedValueRange>\n")
	result.WriteString("<minimum>" + strconv.Itoa(valueRange.Minimum) + "</minimum>\n")
	result.WriteString("<maximum>" + strconv.Itoa(valueRange.Maximum) + "</maximum>\n")
	result.WriteString("<step>" + strconv.Itoa(valueRange.Step) + "</step>\n")
	result.WriteString("</allowedValueRange>\n")

	return result.String()
}

func (value Value) StringXML() string {
	return "<allowedValue>" + strconv.Itoa(value.AllowedValue) + "</allowedValue>"
}
