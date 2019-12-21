package cli

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/micro/protobuf/proto"
	gcli "github.com/urfave/cli"
	"os"

	messages "github.com/fibercrypto/skywallet-protob/go"

	skyWallet "github.com/fibercrypto/skywallet-go/src/skywallet"
)

func featuresCmd() gcli.Command {
	name := "features"
	return gcli.Command{
		Name:         name,
		Usage:        "Ask the device Features.",
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
			msg, err := sq.GetFeatures()
			if err != nil {
				logrus.WithError(err).Errorln("unable to get features")
			} else if msg.Kind == uint16(messages.MessageType_MessageType_Features) {
				features := &messages.Features{}
				if err = proto.Unmarshal(msg.Data, features); err != nil {
					log.Error(err)
					return
				}
				enc := json.NewEncoder(os.Stdout)
				if err = enc.Encode(features); err != nil {
					log.Errorln(err)
					return
				}
				ff := skyWallet.NewFirmwareFeatures(uint64(*features.FirmwareFeatures))
				if err := ff.Unmarshal(); err != nil {
					log.Errorln(err)
					return
				}
				log.Printf("\n\nFirmware features:\n%s", ff)
			} else {
				logrus.Errorln("invalid state")
			}
		},
	}
}
