package cli

import (
	"fmt"
	"github.com/Sirupsen/logrus"
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
			addressIndex := c.Int("addressIndex")
			message := c.String("message")
			walletType := c.String("walletType")
			sq, err := createDevice(c.String("deviceType"))
			if err != nil {
				return
			}
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
