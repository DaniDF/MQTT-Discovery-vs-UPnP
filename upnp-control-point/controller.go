package upnp

import (
	"context"
	"net/url"

	"mobile.dani.df/logging"
	"mobile.dani.df/upnp"

	"github.com/huin/goupnp"
)

func Search(ctx context.Context, st string) (map[string]goupnp.RootDevice, error) {
	log := ctx.Value("logger").(logging.Logger)

	maybeDevices, err := upnp.Search(ctx, st)
	if err != nil {
		log.Error("[upnp-controller] Error while searching for maybeDevices")
		return nil, err
	}

	return search(ctx, maybeDevices)
}

func SearchMx(ctx context.Context, st string, mx int) (map[string]goupnp.RootDevice, error) {
	log := ctx.Value("logger").(logging.Logger)

	maybeDevices, err := upnp.SearchMx(ctx, st, mx)
	if err != nil {
		log.Error("[upnp-controller] Error while searching for maybeDevices")
		return nil, err
	}

	return search(ctx, maybeDevices)
}

func search(ctx context.Context, maybeDevices []upnp.MSearchResult) (map[string]goupnp.RootDevice, error) {
	devices := make(map[string]goupnp.RootDevice)
	for _, maybeDevice := range maybeDevices {
		deviceUrl, err := url.Parse(maybeDevice.Location)
		if err == nil {
			device, err := goupnp.DeviceByURLCtx(ctx, deviceUrl)
			if err == nil {
				devices[device.Device.UDN] = *device
			}

		}
	}

	return devices, nil
}

var subscriptions = make(map[string]string)

func Subscribe(ctx context.Context, rootDevice goupnp.RootDevice, service goupnp.Service, handler func(string)) (*context.CancelFunc, error) {
	log := ctx.Value("logger").(logging.Logger)

	cancelFunc, sid, err := upnp.GenaSubscribeToService(ctx, ConvertRootDevice(rootDevice), ConvertService(service), handler)

	if err == nil {
		subscriptions[rootDevice.Device.UDN+service.ServiceId] = sid
		log.Info("[upnp-controller] Subscribed successfully. Obtained SID: " + sid)
	}

	return cancelFunc, err
}

func Unsubscribe(ctx context.Context, rootDevice goupnp.RootDevice, service goupnp.Service) error {
	return upnp.GenaUnsubscribeFromService(ctx, ConvertRootDevice(rootDevice), ConvertService(service), subscriptions[rootDevice.Device.UDN+service.ServiceId])
}
