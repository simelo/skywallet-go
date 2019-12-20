package proxy //nolint goimports

import (
	"errors"
	"io/ioutil"
	"sync"

	"github.com/fibercrypto/skywallet-go/src/skywallet"
	"github.com/fibercrypto/skywallet-go/src/skywallet/wire"
	messages "github.com/fibercrypto/skywallet-protob/go"
	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/logging"
)

// Sequencer implementation force all messages to be sequential and make the
// command atomic
type Sequencer struct {
	sync.Mutex
	log *logging.MasterLogger
	logCli *logging.MasterLogger
	dev skywallet.Devicer
}

// NewSequencer create a new sequencer instance
func NewSequencer(dev skywallet.Devicer, cliSpeechless bool) skywallet.Devicer {
	sq := &Sequencer{
		log: logging.NewMasterLogger(),
		logCli: logging.NewMasterLogger(),
		dev: dev,
	}
	if cliSpeechless {
		sq.logCli.Out = ioutil.Discard
	}
	return sq
}

func (sq *Sequencer) handleInputInteraction(msg wire.Message) (wire.Message, error) {
	var err error
	handleResponse := func(scopedMsg wire.Message, err error) (string, error) {
		if err != nil {
			sq.log.WithError(err).Errorln("sending message failed" )
			return "", errors.New("sending message failed")
		} else if scopedMsg.Kind == uint16(messages.MessageType_MessageType_Success) {
			msgStr, err := skywallet.DecodeSuccessMsg(scopedMsg)
			if err != nil {
				sq.log.WithError(err).Errorln("unable to decode response")
				return "", errors.New(msgStr)
			}
			return msgStr, nil
		} else if scopedMsg.Kind == uint16(messages.MessageType_MessageType_Failure) {
			msgStr, err := skywallet.DecodeFailMsg(scopedMsg)
			if err != nil {
				sq.log.WithError(err).Errorln("unable to decode response")
				return "", errors.New("unable to decode response")
			}
			return msgStr, nil
		} else {
			return "", errors.New("invalid state")
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) {
		sq.log.Println("PinMatrixRequest request:")
		// FIXME use a reader from sq
		// fmt.Scanln(&pinEnc)
		var pinEnc string
		msg, err = sq.dev.PinMatrixAck(pinEnc)
		msgStr, err := handleResponse(msg, err)
		if err != nil {
			sq.log.WithError(err).Errorln("pin matrixAck ack: sending message failed")
			return msg, err
		}
		sq.logCli.Infof("PinMatrixAck response:", msgStr)
	} else if msg.Kind == uint16(messages.MessageType_MessageType_PassphraseRequest) {
		var passphrase string
		sq.log.Println("PassphraseRequest request:")
		// FIXME use a reader from sq
		//fmt.Scanln(&passphrase)
		msg, err = sq.dev.PassphraseAck(passphrase)
		msgStr, err := handleResponse(msg, err)
		if err != nil {
			sq.log.WithError(err).Errorln("passphrase ack: sending message failed")
			return msg, err
		}
		sq.logCli.Infof("PassphraseAck response:", msgStr)
	} else if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = sq.dev.ButtonAck()
		msgStr, err := handleResponse(msg, err)
		if err != nil {
			sq.log.WithError(err).Errorln("handling message failed")
			return msg, err
		}
		sq.log.Infoln("ButtonAck response", msgStr)
	}
	return msg, nil
}

// AddressGen forward the call to Device and handle all the consecutive command as an
// atomic sequence
func (sq *Sequencer) AddressGen(addressN, startIndex uint32, confirmAddress bool, walletType string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.AddressGen(uint32(addressN), uint32(startIndex), confirmAddress, walletType)
	if err != nil {
		sq.log.WithError(err).Errorln("address gen: sending message failed")
		return wire.Message{}, err
	}
	for msg.Kind != uint16(messages.MessageType_MessageType_ResponseSkycoinAddress) && msg.Kind != uint16(messages.MessageType_MessageType_Failure) {
		if msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) || msg.Kind == uint16(messages.MessageType_MessageType_PassphraseRequest) || msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
			if msg, err = sq.handleInputInteraction(msg); err != nil {
				sq.log.WithError(err).Errorln("error handling interaction")
				return wire.Message{}, err
			}
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_SkycoinAddress) {
		return msg, nil
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_ResponseSkycoinAddress) {
		return msg, nil
	} else if msg.Kind == uint16(messages.MessageType_MessageType_Failure) {
		failMsg, err := skywallet.DecodeFailMsg(msg)
		if err != nil {
			sq.log.WithError(err).Errorln("unable to decode response")
			return wire.Message{}, err
		}
		sq.log.WithError(err).Errorln(failMsg)
		return wire.Message{}, err
	}
	sq.log.Errorln("unexpected message")
	return wire.Message{}, errors.New("unexpected message")
}

// ApplySettings forward the call to Device and handle all the consecutive command as an
// atomic sequence
func (sq *Sequencer) ApplySettings(usePassphrase *bool, label string, language string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.ApplySettings(usePassphrase, label, language)
	if err != nil {
		sq.log.WithError(err).Errorln("apply settings: sending message failed")
		return wire.Message{}, err
	}
	for msg.Kind != uint16(messages.MessageType_MessageType_Failure) && msg.Kind != uint16(messages.MessageType_MessageType_Success) {
		if msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) || msg.Kind == uint16(messages.MessageType_MessageType_PassphraseRequest) {
			if msg, err = sq.handleInputInteraction(msg); err != nil {
				sq.log.WithError(err).Errorln("error handling interaction")
				return wire.Message{}, err
			}
		}
		for msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
			msg, err = sq.dev.ButtonAck()
			if err != nil {
				sq.log.WithError(err).Errorln("unable to apply settings")
				return wire.Message{}, err
			}
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Success) {
		return msg, nil
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Failure) {
		failMsg, err := skywallet.DecodeFailMsg(msg)
		if err != nil {
			sq.log.WithError(err).Errorln("unable to decode response")
			return wire.Message{}, err
		}
		sq.log.WithError(err).Errorln(failMsg)
		return wire.Message{}, err
	}
	sq.log.Errorln("unexpected message")
	return wire.Message{}, errors.New("unexpected message")
}

// Backup forward the call to Device and handle all the consecutive command as an
// atomic sequence
func (sq *Sequencer) Backup() (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.Backup()
	if err != nil {
		sq.log.WithError(err).Errorln("backup: sending message failed")
		return wire.Message{}, err
	}
	for msg.Kind != uint16(messages.MessageType_MessageType_Failure) && msg.Kind != uint16(messages.MessageType_MessageType_Success) {
		if msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) || msg.Kind == uint16(messages.MessageType_MessageType_PassphraseRequest) {
			if msg, err = sq.handleInputInteraction(msg); err != nil {
				sq.log.WithError(err).Errorln("error handling interaction")
				return wire.Message{}, err
			}
		}
		for msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
			msg, err = sq.dev.ButtonAck()
			if err != nil {
				sq.log.WithError(err).Errorln("unable to create backup")
				return wire.Message{}, err
			}
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

// Cancel forward the call to Device
func (sq *Sequencer) Cancel() (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.Cancel()
}

// CheckMessageSignature forward the call to Device
func (sq *Sequencer) CheckMessageSignature(message, signature, address string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.CheckMessageSignature(message, signature, address)
}

// ChangePin forward the call to Device and handle all the consecutive command as an
// atomic sequence
func (sq *Sequencer) ChangePin(removePin *bool) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.ChangePin(new(bool))
	if err != nil {
		sq.log.WithError(err).Errorln("change pin: sending message failed")
		return wire.Message{}, err
	}
	for msg.Kind != uint16(messages.MessageType_MessageType_Failure) && msg.Kind != uint16(messages.MessageType_MessageType_Success) {
		if msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) || msg.Kind == uint16(messages.MessageType_MessageType_PassphraseRequest) || msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
			if msg, err = sq.handleInputInteraction(msg); err != nil {
				sq.log.WithError(err).Errorln("error handling interaction")
				return wire.Message{}, err
			}
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

// Connected forward the call to Device
func (sq *Sequencer) Connected() bool {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.Connected()
}

// Available forward the call to Device
func (sq *Sequencer) Available() bool {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.Available()
}

// FirmwareUpload forward the call to Device
func (sq *Sequencer) FirmwareUpload(payload []byte, hash [32]byte) error {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.FirmwareUpload(payload, hash)
}

// GetFeatures forward the call to Device and handle all the consecutive command as an
// atomic sequence
func (sq *Sequencer) GetFeatures() (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	msg, err := sq.dev.GetFeatures()
	if err != nil {
		sq.log.WithError(err).Errorln("get features: sending message failed")
		return wire.Message{}, err
	}
	for msg.Kind != uint16(messages.MessageType_MessageType_Failure) && msg.Kind != uint16(messages.MessageType_MessageType_Features) {
		if msg.Kind == uint16(messages.MessageType_MessageType_PinMatrixRequest) || msg.Kind == uint16(messages.MessageType_MessageType_PassphraseRequest) || msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
			if msg, err = sq.handleInputInteraction(msg); err != nil {
				sq.log.WithError(err).Errorln("error handling interaction")
				return wire.Message{}, err
			}
		}
	}
	if msg.Kind == uint16(messages.MessageType_MessageType_Features) {
		return msg, nil
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

// GenerateMnemonic forward the call to Device and handle all the consecutive command as an
// atomic sequence
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

// Recovery forward the call to Device and handle all the consecutive command as an
// atomic sequence
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

// SetMnemonic forward the call to Device and handle all the consecutive command as an
// atomic sequence
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

// TransactionSign forward the call to Device and handle all the consecutive command as an
// atomic sequence
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

// SignMessage forward the call to Device and handle all the consecutive command as an
// atomic sequence
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

// Wipe forward the call to Device and handle all the consecutive command as an
// atomic sequence
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

// PinMatrixAck forward the call to Device
func (sq *Sequencer) PinMatrixAck(p string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.PinMatrixAck(p)
}

// WordAck forward the call to Device
func (sq *Sequencer) WordAck(word string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.WordAck(word)
}

// PassphraseAck forward the call to Device
func (sq *Sequencer) PassphraseAck(passphrase string) (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.PassphraseAck(passphrase)
}

// ButtonAck forward the call to Device
func (sq *Sequencer) ButtonAck() (wire.Message, error) {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.ButtonAck()
}

// SetAutoPressButton forward the call to Device
func (sq *Sequencer) SetAutoPressButton(simulateButtonPress bool, simulateButtonType skywallet.ButtonType) error {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.SetAutoPressButton(simulateButtonPress, simulateButtonType)
}

// Close forward the call to Device
func (sq *Sequencer) Close() {
	sq.Lock()
	defer sq.Unlock()
	sq.dev.Close()
}

// Connect forward the call to Device
func (sq *Sequencer) Connect() error {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.Connect()
}

// Disconnect forward the call to Device
func (sq *Sequencer) Disconnect() error {
	sq.Lock()
	defer sq.Unlock()
	return sq.dev.Disconnect()
}
