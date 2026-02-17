package upnp

import (
	"strconv"

	"github.com/huin/goupnp"
	"mobile.dani.df/upnp"
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
		SpecVersion: convertSpecVersion(goupnpRootDevice.SpecVersion),
		Device:      ConvertDevice(goupnpRootDevice.Device),
	}
}

func convertSpecVersion(goupnpSpecVersion goupnp.SpecVersion) upnp.SpecVersion {
	return upnp.SpecVersion{
		Major: strconv.Itoa(int(goupnpSpecVersion.Major)),
		Minor: strconv.Itoa(int(goupnpSpecVersion.Minor)),
	}
}

func ConvertDevice(goupnpDevice goupnp.Device) upnp.Device {
	icons := []upnp.Icon{}
	for _, icon := range goupnpDevice.Icons {
		icons = append(icons, convertIcon(icon))
	}

	services := []upnp.Service{}
	for _, service := range goupnpDevice.Services {
		services = append(services, convertService(service))
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

func convertIcon(goupnpIcon goupnp.Icon) upnp.Icon {
	return upnp.Icon{
		Mimetype: goupnpIcon.Mimetype,
		Height:   strconv.Itoa(int(goupnpIcon.Height)),
		Width:    strconv.Itoa(int(goupnpIcon.Width)),
		Depth:    strconv.Itoa(int(goupnpIcon.Depth)),
		Url:      goupnpIcon.URL.Str,
	}
}

func convertService(goupnpService goupnp.Service) upnp.Service {
	return upnp.Service{
		ServiceType: goupnpService.ServiceType,
		ServiceId:   goupnpService.ServiceId,
		SCPDURL:     goupnpService.SCPDURL.Str,
		EventSubURL: goupnpService.EventSubURL.Str,
		ControlURL:  goupnpService.ControlURL.Str,
	}
}
