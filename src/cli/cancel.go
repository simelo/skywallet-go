package cli

import (
	"fmt"
	gcli "github.com/urfave/cli"

	skyWallet "github.com/fibercrypto/skywallet-go/src/skywallet"
)

func cancelCmd() gcli.Command {
	name := "cancel"
	return gcli.Command{
		Name:         name,
		Usage:        "Ask the device to cancel the ongoing procedure.",
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
			msg, err := sq.Cancel()
			if err != nil {
				log.Error(err)
				return
			}
			responseMsg, err := skyWallet.DecodeSuccessOrFailMsg(msg)
			if err != nil {
				log.Error(err)
				return
			}
			fmt.Println(responseMsg)
		},
	}
}
