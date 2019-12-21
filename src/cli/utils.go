package cli

import (
	"errors"
	"github.com/fibercrypto/skywallet-go/src/integration/proxy"
	"github.com/fibercrypto/skywallet-go/src/skywallet"
	"os"
	"runtime"
)

func parseBool(s string) (*bool, error) {
	var b bool
	switch s {
	case "true":
		b = true
	case "false":
		b = false
	case "":
		return nil, nil
	default:
		return nil, errors.New("Invalid boolean argument")
	}
	return &b, nil
}

func createDevice(devType string) (skywallet.Devicer, error) {
	device := skywallet.NewDevice(skywallet.DeviceTypeFromString(devType))
	if device == nil {
		return nil, errors.New("got null device")
	}
	if os.Getenv("AUTO_PRESS_BUTTONS") == "1" && device.Driver.DeviceType() == skywallet.DeviceTypeEmulator && runtime.GOOS == "linux" {
		err := device.SetAutoPressButton(true, skywallet.ButtonRight)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}
	return proxy.NewSequencer(device, false), nil
}