package upnp

import (
	"strconv"
	"strings"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/scpd"
	"mobile.dani.df/upnp"
	"mobile.dani.df/utils"
)

func ConvertRootDevices(goupnpRootDevices []goupnp.RootDevice) []upnp.RootDevice {
	result := []upnp.RootDevice{}

	for _, rootDevice := range goupnpRootDevices {
		result = append(result, ConvertRootDevice(rootDevice))
	}

	return result
}

func ConvertRootDevice(goupnpRootDevice goupnp.RootDevice) upnp.RootDevice {
	return upnp.RootDevice{
		SpecVersion: ConvertSpecVersion(goupnpRootDevice.SpecVersion),
		URLBase:     goupnpRootDevice.URLBaseStr,
		Device:      ConvertDevice(goupnpRootDevice.Device),
	}
}

func ConvertSpecVersion(goupnpSpecVersion goupnp.SpecVersion) upnp.SpecVersion {
	return upnp.SpecVersion{
		Major: strconv.Itoa(int(goupnpSpecVersion.Major)),
		Minor: strconv.Itoa(int(goupnpSpecVersion.Minor)),
	}
}

func ConvertDevice(goupnpDevice goupnp.Device) upnp.Device {
	icons := []upnp.Icon{}
	for _, icon := range goupnpDevice.Icons {
		icons = append(icons, ConvertIcon(icon))
	}

	services := []upnp.Service{}
	for _, service := range goupnpDevice.Services {
		services = append(services, ConvertService(service))
	}

	devices := []upnp.Device{}
	for _, device := range goupnpDevice.Devices {
		devices = append(devices, ConvertDevice(device))
	}

	return upnp.Device{
		DeviceType:       goupnpDevice.DeviceType,
		UDN:              goupnpDevice.UDN,
		FriendlyName:     goupnpDevice.FriendlyName,
		Manufacturer:     goupnpDevice.Manufacturer,
		ManufacturerURL:  goupnpDevice.ManufacturerURL.Str,
		ModelName:        goupnpDevice.ModelName,
		ModelURL:         goupnpDevice.ModelURL.Str,
		ModelDescription: goupnpDevice.ModelDescription,
		ModelNumber:      goupnpDevice.ModelNumber,
		SerialNumber:     goupnpDevice.SerialNumber,
		UPC:              goupnpDevice.UPC,
		PresentationURL:  goupnpDevice.PresentationURL.Str,
		IconList:         icons,
		ServiceList:      services,
		EmbeddedDevices:  devices,
	}
}

func ConvertIcon(goupnpIcon goupnp.Icon) upnp.Icon {
	return upnp.Icon{
		Mimetype: goupnpIcon.Mimetype,
		Height:   strconv.Itoa(int(goupnpIcon.Height)),
		Width:    strconv.Itoa(int(goupnpIcon.Width)),
		Depth:    strconv.Itoa(int(goupnpIcon.Depth)),
		Url:      goupnpIcon.URL.Str,
	}
}

func ConvertService(goupnpService goupnp.Service) upnp.Service {
	scpd, _ := goupnpService.RequestSCPD() //TODO handle error

	return upnp.Service{
		ServiceType: goupnpService.ServiceType,
		ServiceId:   goupnpService.ServiceId,
		SCPDURL:     goupnpService.SCPDURL.Str,
		EventSubURL: goupnpService.EventSubURL.Str,
		ControlURL:  goupnpService.ControlURL.Str,
		SCPD:        ConvertSCPD(scpd),
	}
}

func ConvertSCPD(s *scpd.SCPD) upnp.Spcd {
	serviceStateTable := []*upnp.StateVariable{}

	for _, stateVariable := range s.StateVariables {
		serviceStateTable = append(serviceStateTable, ConvertSCPDStateVariable(stateVariable))
	}

	result := upnp.Spcd{
		SpecVersion:       ConvertSCPDSpecVersion(s.SpecVersion),
		ServiceStateTable: serviceStateTable,
	}

	for _, action := range s.Actions {
		result.AddAction(ConvertAction(action, serviceStateTable))
	}

	return result
}

func ConvertSCPDSpecVersion(specVersion scpd.SpecVersion) upnp.SpecVersion {
	return upnp.SpecVersion{
		Major: strconv.Itoa(int(specVersion.Major)),
		Minor: strconv.Itoa(int(specVersion.Minor)),
	}
}

func ConvertSCPDStateVariable(stateVariable scpd.StateVariable) *upnp.StateVariable {
	return &upnp.StateVariable{
		SendEvents:   (stateVariable.SendEvents == "yes"),
		Multicast:    (stateVariable.Multicast == "yes"),
		Name:         stateVariable.Name,
		DataType:     stateVariable.DataType.Name,
		DefaultValue: stateVariable.DefaultValue,
		//AllowedValueRange: ConvertAllowedValueRange(*stateVariable.AllowedValueRange),
		AllowedValueList: stateVariable.AllowedValues,
	}
}

func ConvertAllowedValueRange(allowedRange scpd.AllowedValueRange) upnp.ValueRange {
	max, _ := strconv.Atoi(allowedRange.Maximum)
	min, _ := strconv.Atoi(allowedRange.Minimum)
	step, _ := strconv.Atoi(allowedRange.Step)
	return upnp.ValueRange{
		Maximum: max,
		Minimum: min,
		Step:    step,
	}
}

func ConvertAction(action scpd.Action, serviceStateTable []*upnp.StateVariable) upnp.FormalAction {
	argumentList := []upnp.FormalArgument{}

	for _, argument := range action.Arguments {
		argumentList = append(argumentList, ConvertArgument(argument, serviceStateTable))
	}

	return upnp.FormalAction{
		Name:         action.Name,
		ArgumentList: argumentList,
	}
}

func ConvertArgument(argument scpd.Argument, serviceStateTable []*upnp.StateVariable) upnp.FormalArgument {
	direction := upnp.In
	if argument.IsOutput() {
		direction = upnp.Out
	}

	relatedStateVariable, _ := utils.FindFirst(serviceStateTable, func(stateVariable *upnp.StateVariable) bool {
		return stateVariable.Name == argument.RelatedStateVariable
	})

	return upnp.FormalArgument{
		Name:                 argument.Name,
		Direction:            direction,
		RelatedStateVariable: relatedStateVariable,
	}
}

func StringRootDevice(rootDevice goupnp.RootDevice) string {
	var result strings.Builder

	result.WriteString("RootDevice:\n")
	result.WriteString("\t" + strings.ReplaceAll(StringSpecVersion(rootDevice.SpecVersion), "\n", "\n\t"))
	result.WriteString("\n")

	if len(StringUrlBase(rootDevice.URLBaseStr)) > 0 {
		result.WriteString("\t" + strings.ReplaceAll(StringUrlBase(rootDevice.URLBaseStr), "\n", "\n\t"))
		result.WriteString("\n")
	}

	result.WriteString("\t" + strings.ReplaceAll(StringDevice(rootDevice.Device), "\n", "\n\t"))

	return result.String()
}

func StringSpecVersion(specVersion goupnp.SpecVersion) string {
	return "SpecVersion: " + strconv.Itoa(int(specVersion.Major)) + "." + strconv.Itoa(int(specVersion.Minor))
}

func StringUrlBase(urlBase string) string {
	result := ""

	if len(urlBase) > 0 {
		result = "URLBase: " + urlBase
	}

	return result
}

func StringDevice(device goupnp.Device) string {
	var result strings.Builder

	result.WriteString("Device:\n")
	result.WriteString("\tDeviceType: " + device.DeviceType + "\n")
	result.WriteString("\tUDN: " + device.UDN + "\n")
	result.WriteString("\tFriendlyName: " + device.FriendlyName + "\n")
	result.WriteString("\tManufacturer: " + device.Manufacturer + "\n")
	result.WriteString("\tManufacturerURL: " + device.ManufacturerURL.Str + "\n")
	result.WriteString("\tModelName: " + device.ModelName + "\n")
	result.WriteString("\tModelURL: " + device.ModelURL.Str + "\n")
	result.WriteString("\tModelDescription: " + device.ModelDescription + "\n")
	result.WriteString("\tModelNumber: " + device.ModelNumber + "\n")
	result.WriteString("\tSerialNumber: " + device.SerialNumber + "\n")
	result.WriteString("\tUPC: " + device.UPC + "\n")
	result.WriteString("\tPresentationURL: " + device.PresentationURL.Str + "\n")

	for _, icon := range device.Icons {
		result.WriteString("\t" + strings.ReplaceAll(StringIcon(icon), "\n", "\n\t"))
		result.WriteString("\n")
	}

	for _, service := range device.Services {
		result.WriteString("\t" + strings.ReplaceAll(StringService(service), "\n", "\n\t"))
		result.WriteString("\n")
	}

	for _, embeddedDevice := range device.Devices {
		result.WriteString("\t" + strings.ReplaceAll(StringDevice(embeddedDevice), "\n", "\n\t"))
		result.WriteString("\n")
	}

	return result.String()
}

func StringIcon(icon goupnp.Icon) string {
	var result strings.Builder

	result.WriteString("Icon: " + icon.Mimetype)
	result.WriteString(", (" + strconv.Itoa(int(icon.Width)) + "x" + strconv.Itoa(int(icon.Height)) + "x" + strconv.Itoa(int(icon.Depth)) + ")")
	result.WriteString(", " + icon.URL.Str)

	return result.String()
}

func StringService(service goupnp.Service) string {
	var result strings.Builder

	result.WriteString("Service:\n")
	result.WriteString("\tServiceType: " + service.ServiceType + "\n")
	result.WriteString("\tServiceId: " + service.ServiceId + "\n")
	result.WriteString("\tSCPDURL: " + service.SCPDURL.Str + "\n")
	result.WriteString("\tEventSubURL: " + service.EventSubURL.Str + "\n")
	result.WriteString("\tControlURL: " + service.ControlURL.Str + "\n")

	return result.String()
}
