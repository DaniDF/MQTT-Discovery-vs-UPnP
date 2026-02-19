package upnp

import (
	"context"
	"strconv"

	"mobile.dani.df/logging"
	"mobile.dani.df/upnp"

	"github.com/huin/goupnp"
)

func Search(ctx context.Context, st string) (map[string]goupnp.RootDevice, error) {
	log := ctx.Value("logger").(logging.Logger)

	maybeDevices, err := goupnp.DiscoverDevicesCtx(ctx, st)
	if err != nil {
		log.ErrorContext(ctx, "Error occurred while discovering devices")
		return map[string]goupnp.RootDevice{}, err
	}
	log.Debug("Found " + strconv.Itoa(len(maybeDevices)) + " maybe devices")

	devices := make(map[string]goupnp.RootDevice)
	for _, maybeDevice := range maybeDevices {
		if maybeDevice.Err == nil {
			devices[maybeDevice.Root.Device.UDN] = *maybeDevice.Root
		}
	}

	return devices, nil
}

func Subscribe(ctx context.Context, rootDevice goupnp.RootDevice, service goupnp.Service) (*context.CancelFunc, error) {
	return upnp.GenaSubscribeToService(ctx, ConvertRootDevice(rootDevice), ConvertService(service))
}
