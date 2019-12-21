package cli

import (
	"fmt"
	"github.com/Sirupsen/logrus"

	gcli "github.com/urfave/cli"

	messages "github.com/fibercrypto/skywallet-protob/go"

	skyWallet "github.com/fibercrypto/skywallet-go/src/skywallet"
)

func setPinCode() gcli.Command {
	name := "setPinCode"
	return gcli.Command{
		Name:        name,
		Usage:       "Configure a PIN code on a device.",
		Description: "",
		Flags: []gcli.Flag{
			gcli.StringFlag{
				Name:   "deviceType",
				Usage:  "Device type to send instructions to, hardware wallet (USB) or emulator.",
				EnvVar: "DEVICE_TYPE",
			},
		},
		OnUsageError: onCommandUsageError(name),
		Action: func(c *gcli.Context) {
			sq, err := createDevice(c.String("deviceType"))
			if err != nil {
				return
			}
			msg, err := sq.ChangePin(new(bool))
			if err != nil {
				logrus.WithError(err).Errorln("unable to create backup")
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
