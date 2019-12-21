package cli

import (
	"fmt"

	gcli "github.com/urfave/cli"

	messages "github.com/fibercrypto/skywallet-protob/go"

	skyWallet "github.com/fibercrypto/skywallet-go/src/skywallet"
)

func removePinCode() gcli.Command {
	name := "removePinCode"
	return gcli.Command{
		Name:        name,
		Usage:       "Remove a PIN code on a device.",
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
			var pinEnc string
			removePin := new(bool)
			*removePin = true
			sq, err := createDevice(c.String("deviceType"))
			if err != nil {
				return
			}
			msg, err := sq.ChangePin(removePin)
			if err != nil {
				log.Error(err)
				return
			}

			if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
				msg, err = sq.ButtonAck()
				if err != nil {
					log.Error(err)
					return
				}
			}

			for msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) {
				fmt.Printf("PinMatrixRequest response: ")
				fmt.Scanln(&pinEnc)
				msg, err = sq.PinMatrixAck(pinEnc)
				if err != nil {
					log.Error(err)
					return
				}
			}

			// handle success or failure msg
			respMsg, err := skyWallet.DecodeSuccessOrFailMsg(msg)
			if err != nil {
				log.Error(err)
				return
			}

			fmt.Println(respMsg)
		},
	}
}
