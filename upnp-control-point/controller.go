package upnp

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"mobile.dani.df/device-service"
	"mobile.dani.df/logging"
	"mobile.dani.df/upnp"
	"mobile.dani.df/utils"

	"github.com/huin/goupnp"
)

type UpnpController struct {
	ctx context.Context
}

func NewUpnpController() UpnpController {
	ctx := context.Background()
	ctx, _ = logging.Init(ctx, slog.LevelDebug)

	return UpnpController{
		ctx: ctx,
	}
}

// Same as SearchBySt with the default st = "ssdp:all"
func (controller UpnpController) Search() []device.Device {
	return controller.SearchBySt("ssdp:all")
}

func (controller UpnpController) SearchBySt(st string) []device.Device {
	log := controller.ctx.Value("logger").(logging.Logger)

	result := []device.Device{}

	maybeDevices, err := goupnp.DiscoverDevicesCtx(controller.ctx, st)
	if err != nil {
		log.Error("[upnp-controller] Error occurred while discovering devices")
		return result
	}

	log.Debug("[upnp-controller] Found " + strconv.Itoa(len(maybeDevices)) + " maybe devices")

	for _, maybeDevice := range maybeDevices {
		if maybeDevice.Err == nil {
			upnpRootDevice := ConvertRootDevice(*maybeDevice.Root)
			goupnpRootDevice := *maybeDevice.Root

			for _, service := range goupnpRootDevice.Device.Services {
				upnpService := ConvertService(service)
				setAction, find := utils.FindFirst(upnpService.SCPD.GetActions(), func(action upnp.FormalAction) bool {
					return isSetAction(action)
				})

				if find {
					upnpRootDevice.SetStateFunc = func(value string) error {
						type Args struct {
							StateValue string `xml:"StateValue"`
						}
						type Reply struct {
							ActualValue string `xml:"ActualValue"`
						}

						soap := service.NewSOAPClient()
						args := Args{
							StateValue: value,
						}
						reply := Reply{}
						err = soap.PerformActionCtx(controller.ctx, service.ServiceType, setAction.Name, &args, &reply)
						if err != nil {
							log.Error("Error RPC: " + err.Error())
							return err
						}

						return nil
					}

				} else {
					upnpRootDevice.SetStateFunc = func(value string) error {
						return errors.New("Function not defined for this device")
					}
				}

				getAction, find := utils.FindFirst(upnpService.SCPD.GetActions(), func(action upnp.FormalAction) bool {
					return isGetAction(action)
				})

				if find {
					upnpRootDevice.GetStateFunc = func() (string, error) {
						type Args struct{}
						type Reply struct {
							ActualValue string `xml:"ActualValue"`
						}

						soap := service.NewSOAPClient()
						args := Args{}
						reply := Reply{}
						err = soap.PerformActionCtx(controller.ctx, service.ServiceType, getAction.Name, &args, &reply)
						if err != nil {
							log.Error("Error RPC: " + err.Error())
							return "", err
						}

						return reply.ActualValue, nil
					}

				} else {
					upnpRootDevice.GetStateFunc = func() (string, error) {
						return "", errors.New("Function not defined for this device")
					}
				}
			}

			result = append(result, upnpRootDevice)
		}
	}

	return result
}

func Subscribe(ctx context.Context, rootDevice goupnp.RootDevice, service goupnp.Service, handler func(string)) (*context.CancelFunc, error) {
	return upnp.GenaSubscribeToService(ctx, ConvertRootDevice(rootDevice), ConvertService(service), handler)
}

func isSetAction(action upnp.FormalAction) bool {
	return len(action.ArgumentList) == 2 && action.ArgumentList[0].RelatedStateVariable.DataType == "string" && action.ArgumentList[1].RelatedStateVariable.DataType == "string"
}

func isGetAction(action upnp.FormalAction) bool {
	return len(action.ArgumentList) == 1 && action.ArgumentList[0].RelatedStateVariable.DataType == "string"
}
