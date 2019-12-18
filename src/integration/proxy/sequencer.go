package proxy

import (
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/fibercrypto/skywallet-go/src/skywallet"
	"github.com/fibercrypto/skywallet-go/src/skywallet/wire"
	messages "github.com/fibercrypto/skywallet-protob/go"
	"sync"
)

// Sequencer implementation force all messages to be sequential and make the
// command atomic
type Sequencer struct {
	dev skywallet.Devicer
	sync.Mutex
}

func NewSequencer(dev skywallet.Devicer) skywallet.Devicer {
	return &Sequencer{dev:dev}
}

func (sq *Sequencer) AddressGen(addressN, startIndex uint32, confirmAddress bool, walletType string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	var pinEnc string
	msg, err := sq.dev.AddressGen(addressN, startIndex, confirmAddress, walletType)
	if err != nil {
		return wire.Message{}, err
	}
	for msg.Kind != uint16(messages.MessageType_MessageType_ResponseSkycoinAddress) && msg.Kind != uint16(messages.MessageType_MessageType_Failure) {
		if msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) {
			// FIXME use a reader from sq
			// fmt.Printf("PinMatrixRequest response: ")
			// fmt.Scanln(&pinEnc)
			pinAckResponse, err := sq.dev.PinMatrixAck(pinEnc)
			if err != nil {
				return wire.Message{}, err
			}
			// TODO this log
			logrus.Errorln("PinMatrixAck response: %s", pinAckResponse)
			continue
		}

		if msg.Kind == uint16(messages.MessageType_MessageType_PassphraseRequest) {
			var passphrase string
			// FIXME use a reader from sq
			//fmt.Printf("Input passphrase: ")
			//fmt.Scanln(&passphrase)
			passphraseAckResponse, err := sq.dev.PassphraseAck(passphrase)
			if err != nil {
				return wire.Message{}, nil
			}
			// TODO this log
			logrus.Errorln("PinMatrixAck response: %s", passphraseAckResponse)
			continue
		}

		if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
			msg, err = sq.dev.ButtonAck()
			if err != nil {
				return wire.Message{}, err
			}
			continue
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_ResponseSkycoinAddress) {
		return msg, nil
	}
	failMsg, err := skywallet.DecodeFailMsg(msg)
	if err != nil {
		return wire.Message{}, err
	}
	logrus.WithError(err).Errorln(failMsg)
	return wire.Message{}, err
}

func (sq *Sequencer) ApplySettings(usePassphrase *bool, label string, language string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.ApplySettings(usePassphrase, label, language)
	if err != nil {
		return wire.Message{}, nil
	}
	for msg.Kind != uint16(messages.MessageType_MessageType_Failure) && msg.Kind != uint16(messages.MessageType_MessageType_Success) {
		if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
			msg, err = sq.dev.ButtonAck()
			if err != nil {
				return wire.Message{}, err
			}
			continue
		}
		if msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) {
			var pinEnc string
			// FIXME use a reader from sq
			//fmt.Printf("PinMatrixRequest response: ")
			//fmt.Scanln(&pinEnc)
			/*pinAckResponse*/_, err := sq.dev.PinMatrixAck(pinEnc)
			if err != nil {
				return wire.Message{}, err
			}
			// log.Infof("PinMatrixAck response: %s", pinAckResponse)
			continue
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Failure) {
		failMsg, err := skywallet.DecodeFailMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.WithError(err).Errorln(failMsg)
		return wire.Message{}, err
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Success) {
		successMsg, err := skywallet.DecodeSuccessMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.Info(successMsg)
		return msg, nil
	}
	logrus.WithField("msg", msg).Errorln("unexpected response from device")
	return wire.Message{}, errors.New("unexpected response from device")
}

func (sq *Sequencer) Backup() (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.Backup()
	if err != nil {
		return wire.Message{}, err
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) {
		// FIXME use a reader from sq
		var pinEnc string
		//fmt.Printf("PinMatrixRequest response: ")
		//fmt.Scanln(&pinEnc)
		msg, err := sq.dev.PinMatrixAck(pinEnc)
		if err != nil {
			return wire.Message{}, nil
		}
		for msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
			msg, err = sq.dev.ButtonAck()
			if err != nil {
				return wire.Message{}, err
			}
		}
	}
	for msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = sq.dev.ButtonAck()
		if err != nil {
			return wire.Message{}, err
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Success) {
		msgStr, err := skywallet.DecodeSuccessMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.Info(msgStr)
		return msg, nil
	}
	responseMsg, err := skywallet.DecodeFailMsg(msg)
	if err != nil {
		return wire.Message{}, err
	}
	logrus.WithError(err).Errorln(responseMsg)
	return wire.Message{}, errors.New("error in backup operation")
}

func (sq *Sequencer) Cancel() (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.Cancel()
}

func (sq *Sequencer) CheckMessageSignature(message, signature, address string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.CheckMessageSignature(message, signature, address)
}

func (sq *Sequencer) ChangePin(removePin *bool) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	var pinEnc string
	msg, err := sq.dev.ChangePin(new(bool))
	if err != nil {
		return wire.Message{}, err
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = sq.dev.ButtonAck()
		if err != nil {
			return wire.Message{}, err
		}
	}
	for msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) {
		// FIXME use a reader from sq
		//fmt.Printf("PinMatrixRequest response: ")
		//fmt.Scanln(&pinEnc)
		msg, err = sq.dev.PinMatrixAck(pinEnc)
		if err != nil {
			return wire.Message{}, err
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Success) {
		respMsg, err := skywallet.DecodeSuccessMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.WithError(err).Errorln(respMsg)
		return wire.Message{}, nil
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Failure) {
		respMsg, err := skywallet.DecodeFailMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.WithError(err).Errorln(respMsg)
		return wire.Message{}, errors.New(respMsg)
	}
	logrus.WithField("msg", msg).Errorln("unexpected response from device")
    return wire.Message{}, errors.New("unexpected response from device")
}

func (sq *Sequencer) Connected() bool {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.Connected()
}

func (sq *Sequencer) Available() bool {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.Available()
}

func (sq *Sequencer) FirmwareUpload(payload []byte, hash [32]byte) error {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.FirmwareUpload(payload, hash)
}

func (sq *Sequencer) GetFeatures() (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.GetFeatures()
	if err != nil {
		return wire.Message{}, err
	}
	switch msg.Kind {
	case uint16(messages.MessageType_MessageType_Features):
		return msg, nil
	case uint16(messages.MessageType_MessageType_Failure), uint16(messages.MessageType_MessageType_Success):
		msgData, err := skywallet.DecodeFailMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.Errorln(msgData)
		return wire.Message{}, err
	default:
		logrus.Errorf("received unexpected message type: %s", messages.MessageType(msg.Kind))
		return wire.Message{}, errors.New("unexpected msg")
	}
}

func (sq *Sequencer) GenerateMnemonic(wordCount uint32, usePassphrase bool) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.GenerateMnemonic(wordCount, usePassphrase)
	if err != nil {
		return wire.Message{}, err
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = sq.dev.ButtonAck()
		if err != nil {
			return wire.Message{}, err
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Success) {
		_, err := skywallet.DecodeSuccessMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		return msg, nil
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Failure) {
		msgStr, err := skywallet.DecodeFailMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.WithError(err).Errorln(msgStr)
		return wire.Message{}, errors.New(msgStr)
	}
	logrus.WithField("msg", msg).Errorln("unexpected response from device")
	return wire.Message{}, errors.New("unexpected response from device")
}

func (sq *Sequencer) Recovery(wordCount uint32, usePassphrase *bool, dryRun bool) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.Recovery(wordCount, usePassphrase, dryRun)
	if err != nil {
		return wire.Message{}, err
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = sq.dev.ButtonAck()
		if err != nil {
			return wire.Message{}, err
		}
	}
	for msg.Kind == uint16(messages.MessageType_MessageType_WordRequest) {
		// FIXME use a reader from sq
		var word string
		//fmt.Printf("Word: ")
		//fmt.Scanln(&word)
		msg, err = sq.dev.WordAck(word)
		if err != nil {
			return wire.Message{}, err
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = sq.dev.ButtonAck()
		if err != nil {
			return wire.Message{}, err
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Failure) {
		msgStr, err := skywallet.DecodeFailMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.WithError(err).Errorln(msgStr)
		return wire.Message{}, err
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Success) {
		_, err := skywallet.DecodeSuccessMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		return msg, nil
	}
	logrus.WithField("msg", msg).Errorln("unexpected response from device")
	return wire.Message{}, errors.New("unexpected response from device")
}

func (sq *Sequencer) SetMnemonic(mnemonic string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.SetMnemonic(mnemonic)
	if err != nil {
		return wire.Message{}, err
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = sq.dev.ButtonAck()
		if err != nil {
			return wire.Message{}, err
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Failure) {
		msgStr, err := skywallet.DecodeFailMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.WithError(err).Errorln(msgStr)
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Success) {
		_, err := skywallet.DecodeSuccessMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		return msg, nil
	}
	logrus.WithField("msg", msg).Errorln("unexpected response from device")
	return wire.Message{}, errors.New("unexpected response from device")
}

func (sq *Sequencer) TransactionSign(inputs []*messages.SkycoinTransactionInput, outputs []*messages.SkycoinTransactionOutput, walletType string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	// TODO interesting
	//if len(inputs) != len(inputIndex) {
	//	fmt.Println("Every given input hash should have the an inputIndex")
	//	return
	//}
	//if len(outputs) != len(coins) || len(outputs) != len(hours) {
	//	fmt.Println("Every given output should have a coin and hour value")
	//	return
	//}
	msg, err := sq.dev.TransactionSign(inputs, outputs, walletType)
	if err != nil {
		return wire.Message{}, err
	}
	for {
		logrus.Errorln("msg.Kind", msg.Kind)
		switch msg.Kind {
		case uint16(messages.MessageType_MessageType_ResponseTransactionSign):
			return msg, nil
		case uint16(messages.MessageType_MessageType_Success):
			return wire.Message{}, errors.New("should end with ResponseTransactionSign request")
		case uint16(messages.MessageType_MessageType_ButtonRequest):
			msg, err = sq.dev.ButtonAck()
			if err != nil {
				return wire.Message{}, err
			}
		case uint16(messages.MessageType_MessageType_PassphraseRequest):
			var passphrase string
			// FIXME use a reader from sq
			//fmt.Printf("Input passphrase: ")
			//fmt.Scanln(&passphrase)
			msg, err = sq.dev.PassphraseAck(passphrase)
			if err != nil {
				return wire.Message{}, err
			}
		case uint16(messages.MessageType_MessageType_PinMatrixRequest):
			var pinEnc string
			// FIXME use a reader from sq
			//fmt.Printf("PinMatrixRequest response: ")
			//fmt.Scanln(&pinEnc)
			msg, err = sq.dev.PinMatrixAck(pinEnc)
			if err != nil {
				return wire.Message{}, err
			}
		case uint16(messages.MessageType_MessageType_Failure):
			failMsg, err := skywallet.DecodeFailMsg(msg)
			if err != nil {
				return wire.Message{}, err
			}
			logrus.WithError(err).Errorln(failMsg)
			return wire.Message{}, errors.New("failed")
		default:
			logrus.Errorf("received unexpected message type: %s", messages.MessageType(msg.Kind))
			return wire.Message{}, errors.New("unexpected message")
		}
	}
}

func (sq *Sequencer) SignMessage(addressN, addressIndex int, message string, walletType string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.SignMessage(1, addressIndex, message, walletType)
	if err != nil {
		return wire.Message{}, err
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = sq.dev.ButtonAck()
		if err != nil {
			return wire.Message{}, err
		}
	}
	for msg.Kind != uint16(messages.MessageType_MessageType_ResponseSkycoinSignMessage) && msg.Kind != uint16(messages.MessageType_MessageType_Failure) {
		if msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) {
			var pinEnc string
			// FIXME use a reader from sq
			//fmt.Printf("PinMatrixRequest response: ")
			//fmt.Scanln(&pinEnc)
			msg, err = sq.dev.PinMatrixAck(pinEnc)
			if err != nil {
				return wire.Message{}, err
			}
			continue
		}
		if msg.Kind == uint16(messages.MessageType_MessageType_PassphraseRequest) {
			var passphrase string
			// FIXME use a reader from sq
			//fmt.Printf("Input passphrase: ")
			//fmt.Scanln(&passphrase)
			msg, err = sq.dev.PassphraseAck(passphrase)
			if err != nil {
				return wire.Message{}, err
			}
			continue
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_ResponseSkycoinSignMessage) {
		_, err = skywallet.DecodeResponseSkycoinSignMessage(msg)
		if err != nil {
			return wire.Message{}, err
		}
		return msg, nil
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Failure) {
		msgStr, err := skywallet.DecodeFailMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.WithError(err).Errorln(msgStr)
		return wire.Message{}, err
	}
	logrus.WithField("msg", msg).Errorln("unexpected response from device")
	return wire.Message{}, errors.New("unexpected response from device")
}

func (sq *Sequencer) Wipe() (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.Wipe()
	if err != nil {
		return wire.Message{}, err
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = sq.dev.ButtonAck()
		if err != nil {
			return wire.Message{}, err
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Success) {
		_, err := skywallet.DecodeSuccessMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		return msg, nil
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Failure) {
		msgStr, err := skywallet.DecodeFailMsg(msg)
		if err != nil {
			return wire.Message{}, err
		}
		logrus.WithError(err).Errorln(msgStr)
		return wire.Message{}, err
	}
	logrus.WithField("msg", msg).Errorln("unexpected response from device")
	return wire.Message{}, errors.New("unexpected response from device")
}

func (sq *Sequencer) PinMatrixAck(p string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.PinMatrixAck(p)
}

func (sq *Sequencer) WordAck(word string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.WordAck(word)
}

func (sq *Sequencer) PassphraseAck(passphrase string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.PassphraseAck(passphrase)
}

func (sq *Sequencer) ButtonAck() (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.ButtonAck()
}

func (sq *Sequencer) SetAutoPressButton(simulateButtonPress bool, simulateButtonType skywallet.ButtonType) error {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.SetAutoPressButton(simulateButtonPress, simulateButtonType)
}

func (sq *Sequencer) Close() {
	sq.Lock()
	defer sq.Unlock()
	sq.dev.Close()
}

func (sq *Sequencer) Connect() error {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.Connect()
}

func (sq *Sequencer) Disconnect() error {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.Disconnect()
}