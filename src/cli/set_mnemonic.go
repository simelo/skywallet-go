package cli

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	messages "github.com/fibercrypto/skywallet-protob/go"

	gcli "github.com/urfave/cli"

	skyWallet "github.com/fibercrypto/skywallet-go/src/skywallet"
)

func setMnemonicCmd() gcli.Command {
	name := "setMnemonic"
	return gcli.Command{
		Name:        name,
		Usage:       "Configure the device with a mnemonic.",
		Description: "",
		Flags: []gcli.Flag{
			gcli.StringFlag{
				Name:  "mnemonic",
				Usage: "Mnemonic that will be stored in the device to generate addresses.",
			},
			gcli.StringFlag{
				Name:   "deviceType",
				Usage:  "Device type to send instructions to, hardware wallet (USB) or emulator.",
				EnvVar: "DEVICE_TYPE",
			},
		},
		OnUsageError: onCommandUsageError(name),
		Action: func(c *gcli.Context) {
			mnemonic := c.String("mnemonic")
			sq, err := createDevice(c.String("deviceType"))
			if err != nil {
				return
			}
			msg, err := sq.SetMnemonic(mnemonic)
			if err != nil {
				logrus.WithError(err).Errorln("unable to set mnemonic")
			} else if msg.Kind == uint16(messages.MessageType_MessageType_Success) {
				msgStr, err := skyWallet.DecodeSuccessMsg(msg)
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
