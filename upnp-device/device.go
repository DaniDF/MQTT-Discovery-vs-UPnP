package upnp

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
