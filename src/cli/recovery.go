package cli

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	gcli "github.com/urfave/cli"

	messages "github.com/fibercrypto/skywallet-protob/go"

	skyWallet "github.com/fibercrypto/skywallet-go/src/skywallet"
)

func recoveryCmd() gcli.Command {
	name := "recovery"
	return gcli.Command{
		Name:        name,
		Usage:       "Ask the device to perform the seed recovery procedure.",
		Description: "",
		Flags: []gcli.Flag{
			gcli.StringFlag{
				Name:  "usePassphrase",
				Usage: "Configure a passphrase",
			},
			gcli.BoolFlag{
				Name:  "dryRun",
				Usage: "perform dry-run recovery workflow (for safe mnemonic validation)",
			},
			gcli.IntFlag{
				Name:  "wordCount",
				Usage: "Use a specific (12 | 24) number of words for the Mnemonic recovery",
				Value: 12,
			},
			gcli.StringFlag{
				Name:   "deviceType",
				Usage:  "Device type to send instructions to, hardware wallet (USB) or emulator.",
				EnvVar: "DEVICE_TYPE",
			},
		},
		OnUsageError: onCommandUsageError(name),
		Action: func(c *gcli.Context) {
			passphrase := c.String("usePassphrase")
			usePassphrase, _err := parseBool(passphrase)
			if _err != nil {
				log.Errorln("Valid values for usePassphrase are true or false")
				return
			}
			dryRun := c.Bool("dryRun")
			wordCount := uint32(c.Uint64("wordCount"))
			sq, err := createDevice(c.String("deviceType"))
			if err != nil {
				return
			}
			msg, err := sq.Recovery(wordCount, usePassphrase, dryRun)
			if err != nil {
				logrus.WithError(err).Errorln("unable to recover device")
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
