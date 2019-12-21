package cli

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	messages "github.com/fibercrypto/skywallet-protob/go"

	gcli "github.com/urfave/cli"

	skyWallet "github.com/fibercrypto/skywallet-go/src/skywallet"
)

func wipeCmd() gcli.Command {
	name := "wipe"
	return gcli.Command{
		Name:         name,
		Usage:        "Ask the device to wipe clean all the configuration it contains.",
		Description:  "",
		OnUsageError: onCommandUsageError(name),
		Flags: []gcli.Flag{
			gcli.StringFlag{
				Name:   "deviceType",
				Usage:  "Device type to send instructions to, hardware wallet (USB) or emulator.",
				EnvVar: "DEVICE_TYPE",
			},
		},
		Action: func(c *gcli.Context) {
			sq, err := createDevice(c.String("deviceType"))
			if err != nil {
				return
			}
			msg, err := sq.Wipe()
			if err != nil {
				logrus.WithError(err).Errorln("unable to wipe the device")
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
