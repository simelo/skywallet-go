package cli

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/fibercrypto/skywallet-go/src/integration/proxy"
	"os"
	"runtime"

	messages "github.com/fibercrypto/skywallet-protob/go"

	gcli "github.com/urfave/cli"

	skyWallet "github.com/fibercrypto/skywallet-go/src/skywallet"
)

func addressGenCmd() gcli.Command {
	name := "addressGen"
	return gcli.Command{
		Name:        name,
		Usage:       "Generate skycoin addresses using the firmware",
		Description: "",
		Flags: []gcli.Flag{
			gcli.IntFlag{
				Name:  "addressN",
				Value: 1,
				Usage: "Number of addresses to generate. Assume 1 if not set.",
			},
			gcli.IntFlag{
				Name:  "startIndex",
				Value: 0,
				Usage: "Index where deterministic key generation will start from. Assume 0 if not set.",
			},
			gcli.BoolFlag{
				Name:  "confirmAddress",
				Usage: "If requesting one address it will be sent only if user confirms operation by pressing device's button.",
			},
			gcli.StringFlag{
				Name:   "deviceType",
				Usage:  "Device type to send instructions to, hardware wallet (USB) or emulator.",
				EnvVar: "DEVICE_TYPE",
			},
			gcli.StringFlag{
				Name:     "walletType",
				Usage:    "Wallet type. Types are \"deterministic\" or \"bip44\"",
				Required: true,
			},
		},
		OnUsageError: onCommandUsageError(name),
		Action: func(c *gcli.Context) {
			walletType := c.String("walletType")
			addressN := c.Int("addressN")
			startIndex := c.Int("startIndex")
			confirmAddress := c.Bool("confirmAddress")
			device := skyWallet.NewDevice(skyWallet.DeviceTypeFromString(c.String("deviceType")))
			if device == nil {
				return
			}
			if os.Getenv("AUTO_PRESS_BUTTONS") == "1" && device.Driver.DeviceType() == skyWallet.DeviceTypeEmulator && runtime.GOOS == "linux" {
				err := device.SetAutoPressButton(true, skyWallet.ButtonRight)
				if err != nil {
					log.Error(err)
					return
				}
			}
			sq := proxy.NewSequencer(device, false)
			msg, err := sq.AddressGen(uint32(addressN), uint32(startIndex), confirmAddress, walletType)
			if err != nil {
				logrus.WithError(err).Errorln("unable to get address")
			} else if msg.Kind == uint16(messages.MessageType_MessageType_ResponseSkycoinAddress) {
				msgStr, err := skyWallet.DecodeResponseSkycoinAddress(msg)
				if err != nil {
					logrus.WithError(err).Errorln("unable to get address")
					return
				}
				fmt.Println(msgStr)
			} else {
				logrus.Errorln("invalid state")
			}
		},
	}
}
