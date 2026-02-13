package upnp

import "strings"

/*
<root xmlns="urn:schemas-upnp-org:device-1-0" xmlns:pnpx="http://schemas.microsoft.com/windows/pnpx/2005/11" xmlns:df="http://schemas.microsoft.com/windows/2008/09/devicefoundation">
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
	</device>
</root>
*/

type RootDevice struct {
	SpecVersion SpecVersion
	Devices     []Device
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
}

type Icon struct {
	Mimetype string
	Height   string
	Width    string
	Depth    string
	Url      string
}

type Service struct {
	ServiceType string
	ServiceId   string
	SCPDURL     string
	EventSubURL string
	ControlURL  string
}

func (rootDevice RootDevice) StringXML() string {
	var result strings.Builder

	result.WriteString("<root xmlns=\"urn:schemas-upnp-org:device-1-0\" xmlns:pnpx=\"http://schemas.microsoft.com/windows/pnpx/2005/11\" xmlns:df=\"http://schemas.microsoft.com/windows/2008/09/devicefoundation\">\n")
	result.WriteString(rootDevice.SpecVersion.StringXML())

	for _, device := range rootDevice.Devices {
		result.WriteString(device.StringXML())
	}

	result.WriteString("</root>\n")

	return result.String()
}

func (specVersion SpecVersion) StringXML() string {
	var result strings.Builder

	result.WriteString("<specVersion>\n")
	result.WriteString("<major>" + specVersion.Major + "</major>\n")
	result.WriteString("<minor>" + specVersion.Minor + "</minor>\n")
	result.WriteString("</specVersion>\n")

	return result.String()
}

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

	result.WriteString("</device>\n")

	return result.String()
}

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
