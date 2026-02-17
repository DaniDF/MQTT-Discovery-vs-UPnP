package upnp

import (
	"context"
	"strconv"

	"mobile.dani.df/logging"

	"github.com/huin/goupnp"
)

func Search(ctx context.Context, st string) ([]goupnp.RootDevice, error) {
	log := ctx.Value("logger").(logging.Logger)

	maybeDevices, err := goupnp.DiscoverDevicesCtx(ctx, st)
	if err != nil {
		log.ErrorContext(ctx, "Error occurred while discovering devices")
		return []goupnp.RootDevice{}, err
	}
	log.Debug("Found " + strconv.Itoa(len(maybeDevices)) + " maybe devices")

	devices := []goupnp.RootDevice{}
	for _, maybeDevice := range maybeDevices {
		if maybeDevice.Err == nil {
			devices = append(devices, *maybeDevice.Root)
		}
	}

	return devices, nil
}
