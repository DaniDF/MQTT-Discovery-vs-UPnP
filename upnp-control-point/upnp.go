package upnp

import (
	"context"
	"fmt"
	"strconv"

	"mobile.dani.df/logging"

	"github.com/huin/goupnp"
)

func Scan(ctx context.Context, filter string) {
	log := ctx.Value("logger").(logging.Logger)
	maybeDevices, err := goupnp.DiscoverDevicesCtx(ctx, filter)

	if err != nil {
		log.ErrorContext(ctx, "Error occurred while discovering devices")
	}

	devices := []goupnp.MaybeRootDevice{}
	fmt.Println("--- Maybe Device list ---")
	for i, maybeDevice := range maybeDevices {
		log.DebugContext(ctx, maybeDevice.USN)
		fmt.Println(strconv.Itoa(i) + ") " + maybeDevice.USN)

		if maybeDevice.Err == nil {
			devices = append(devices, maybeDevice)
		}
	}
	fmt.Println("-------------------------")

	fmt.Println("------ Device list ------")
	for i, device := range devices {
		log.DebugContext(ctx, device.Root.Device.UDN)
		fmt.Println(strconv.Itoa(i) + ") " + device.Root.Device.UDN)
		fmt.Println("\tURL: " + device.Root.URLBase.String())
		fmt.Println("\tXML Scheme: " + device.Root.XMLName.Local + " - " + device.Root.XMLName.Space)
		fmt.Println("\tDevice type: " + device.Root.Device.DeviceType)
		fmt.Println("\tFriendly name: " + device.Root.Device.FriendlyName)
		fmt.Println("\tManifacturer: " + device.Root.Device.Manufacturer)
		fmt.Println("\tManifacturer URL: " + device.Root.Device.ManufacturerURL.Str)
		fmt.Println("\tModel description: " + device.Root.Device.ModelDescription)
		fmt.Println("\tModel name: " + device.Root.Device.ModelName)
		fmt.Println("\tModel number: " + device.Root.Device.ModelNumber)
		fmt.Println("\tModel URL: " + device.Root.Device.ModelURL.Str)
		fmt.Println("\tUPC: " + device.Root.Device.UPC)
		fmt.Println("\tPresentationURL: " + device.Root.Device.PresentationURL.Str)

		fmt.Println("\tIcons:")
		for _, icon := range device.Root.Device.Icons {
			fmt.Println("\t\t" + icon.URL.Str)
		}

		fmt.Println("\tServices:")
		for _, service := range device.Root.Device.Services {
			fmt.Println("\t\tId: " + service.ServiceId)
			fmt.Println("\t\t\tService type: " + service.ServiceType)
			fmt.Println("\t\t\tSCPDURL: " + service.SCPDURL.Str)
			fmt.Println("\t\t\tControl URL: " + service.ControlURL.Str)
			fmt.Println("\t\t\tEvent sub URL: " + service.EventSubURL.Str)
		}

		fmt.Println("\tDevices: ")
		for _, device := range device.Root.Device.Devices {
			fmt.Println("\t\t" + device.String())
		}
	}
	fmt.Println("-------------------------")
}
