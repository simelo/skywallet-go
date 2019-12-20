package cli

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/fibercrypto/skywallet-go/src/integration/proxy"
	"os"
	"runtime"

	gcli "github.com/urfave/cli"

	messages "github.com/fibercrypto/skywallet-protob/go"

	skyWallet "github.com/fibercrypto/skywallet-go/src/skywallet"
)

func signMessageCmd() gcli.Command {
	name := "signMessage"
	return gcli.Command{
		Name:        name,
		Usage:       "Ask the device to sign a message using the secret key at given index.",
		Description: "",
		Flags: []gcli.Flag{
			gcli.IntFlag{
				Name:  "addressIndex",
				Value: 0,
				Usage: "Index of the address that will issue the signature. Assume 0 if not set.",
			},
			gcli.StringFlag{
				Name:  "message",
				Usage: "The message that the signature claims to be signing.",
			},
			gcli.StringFlag{
				Name:   "deviceType",
				Usage:  "Device type to send instructions to, hardware wallet (USB) or emulator.",
				EnvVar: "DEVICE_TYPE",
			},
			gcli.StringFlag{
				Name:  "walletType",
				Usage: "Wallet type. Types are \"deterministic\" or \"bip44\"",
			},
		},
		OnUsageError: onCommandUsageError(name),
		Action: func(c *gcli.Context) {
			device := skyWallet.NewDevice(skyWallet.DeviceTypeFromString(c.String("deviceType")))
			if device == nil {
				return
			}
			defer device.Close()

			if os.Getenv("AUTO_PRESS_BUTTONS") == "1" && device.Driver.DeviceType() == skyWallet.DeviceTypeEmulator && runtime.GOOS == "linux" {
				err := device.SetAutoPressButton(true, skyWallet.ButtonRight)
				if err != nil {
					log.Error(err)
					return
				}
			}

			addressIndex := c.Int("addressIndex")
			message := c.String("message")
			walletType := c.String("walletType")
			sq := proxy.NewSequencer(device, false)
			msg, err := sq.SignMessage(1, addressIndex, message, walletType)
			if err != nil {
				logrus.WithError(err).Errorln("unable to sign transaction")
			} else if msg.Kind == uint16(messages.MessageType_MessageType_ResponseSkycoinSignMessage) {
				msgStr, err := skyWallet.DecodeResponseSkycoinSignMessage(msg)
				if err != nil {
					logrus.WithError(err).Errorln("unable to decode response")
					return
				}
				fmt.Println(msgStr)
			} else {
				logrus.Errorln("invalid state")
			}
		},
	}
}
