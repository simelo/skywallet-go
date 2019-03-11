package devicewallet

import (
	"errors"
	"io"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	messages "github.com/skycoin/hardware-wallet-go/src/device-wallet/messages/go"
	"github.com/skycoin/hardware-wallet-go/src/device-wallet/wire"
)

// DeviceType type of device: emulator or usb
type DeviceType int32

var (
	log = logging.MustGetLogger("device-wallet")
)

const (
	// DeviceTypeEmulator use emulator
	DeviceTypeEmulator DeviceType = 1
	// DeviceTypeUsb use usb
	DeviceTypeUSB DeviceType = 2
)

// Devicer provides api for the hw wallet functions
type Devicer interface {
	AddressGen(addressN, startIndex int, confirmAddress bool) (wire.Message, error)
	ApplySettings(usePassphrase bool, label string) (wire.Message, error)
	Backup() (wire.Message, error)
	Cancel() (wire.Message, error)
	CheckMessageSignature(message, signature, address string) (wire.Message, error)
	ChangePin() (wire.Message, error)
	Connected() bool
	FirmwareUpload(payload []byte, hash [32]byte) error
	GetFeatures() (wire.Message, error)
	GenerateMnemonic(wordCount uint32, usePassphrase bool) (wire.Message, error)
	Recovery(wordCount uint32, usePassphrase, dryRun bool) (wire.Message, error)
	SetMnemonic(mnemonic string) (wire.Message, error)
	TransactionSign(inputs []*messages.SkycoinTransactionInput, outputs []*messages.SkycoinTransactionOutput) (wire.Message, error)
	SignMessage(addressN int, message string) (wire.Message, error)
	Wipe() (wire.Message, error)
	PinMatrixAck(p string) (wire.Message, error)
	WordAck(word string) (wire.Message, error)
	PassphraseAck(passphrase string) (wire.Message, error)
	ButtonAck() (wire.Message, error)
}

// Device provides hardware wallet functions
type Device struct {
	DeviceType
}

// NewEmulatorDevice returns an emulator device
func NewEmulatorDevice() *Device {
	return &Device{
		DeviceTypeEmulator,
	}
}

// NewUSBDevice returns a USB device
func NewUSBDevice() *Device {
	return &Device{
		DeviceTypeUSB,
	}
}

// AddressGen Ask the device to generate an address
func (d *Device) AddressGen(addressN, startIndex int, confirmAddress bool) (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	chunks, err := MessageAddressGen(addressN, startIndex, confirmAddress)
	if err != nil {
		return wire.Message{}, err
	}

	return sendToDevice(dev, chunks)
}

// ApplySettings send ApplySettings request to the device
func (d *Device) ApplySettings(usePassphrase bool, label string) (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	chunks, err := MessageApplySettings(usePassphrase, label)
	if err != nil {
		return wire.Message{}, err
	}

	return sendToDevice(dev, chunks)
}

// BackupDevice ask the device to perform the seed backup
func (d *Device) Backup() (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()
	var msg wire.Message

	var chunks [][64]byte
	err = initialize(dev)
	if err != nil {
		return wire.Message{}, err
	}

	chunks, err = MessageBackup()
	if err != nil {
		return wire.Message{}, err
	}

	msg, err = sendToDevice(dev, chunks)
	if err != nil {
		return wire.Message{}, err
	}

	for msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = deviceButtonAck(dev)
		if err != nil {
			return wire.Message{}, err
		}
	}

	return msg, nil
}

// Cancel send Cancel request
func (d *Device) Cancel() (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	chunks, err := MessageCancel()
	if err != nil {
		return wire.Message{}, err
	}

	return sendToDevice(dev, chunks)
}

// CheckMessageSignature Check a message signature matches the given address.
func (d *Device) CheckMessageSignature(message, signature, address string) (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	// Send CheckMessageSignature
	chunks, err := MessageCheckMessageSignature(message, signature, address)
	if err != nil {
		return wire.Message{}, err
	}

	return sendToDevice(dev, chunks)
}

// ChangePin changes device's PIN code
// The message that is sent contains an encoded form of the PIN.
// The digits of the PIN are displayed in a 3x3 matrix on the Trezor,
// and the message that is sent back is a string containing the positions
// of the digits on that matrix. Below is the mapping between positions
// and characters to be sent:
// 7 8 9
// 4 5 6
// 1 2 3
// For example, if the numbers are laid out in this way on the Trezor,
// 3 1 5
// 7 8 4
// 9 6 2
// To set the PIN "12345", the positions are:
// top, bottom-right, top-left, right, top-right
// so you must send "83769".
func (d *Device) ChangePin() (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	chunks, err := MessageChangePin()
	if err != nil {
		return wire.Message{}, err
	}

	msg, err := sendToDevice(dev, chunks)
	if err != nil {
		return wire.Message{}, err
	}

	// Acknowledge that a button has been pressed
	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = deviceButtonAck(dev)
		if err != nil {
			return msg, err
		}
	}
	return msg, nil
}

// Connected check if a device is connected
func (d *Device) Connected() bool {
	return deviceConnected(d.DeviceType)
}

// FirmwareUpload Updates device's firmware
func (d *Device) FirmwareUpload(payload []byte, hash [32]byte) error {
	if d.DeviceType != DeviceTypeUSB {
		return errors.New("wrong device type")
	}

	dev, err := d.GetDevice()
	if err != nil {
		return err
	}
	defer dev.Close()

	err = initialize(dev)
	if err != nil {
		return err
	}

	log.Printf("Length of firmware %d", uint32(len(payload)))

	chunks, err := MessageFirmwareErase(payload)
	if err != nil {
		return err
	}
	erasemsg, err := sendToDevice(dev, chunks)
	if err != nil {
		return err
	}
	log.Printf("Success %d! FirmwareErase %s\n", erasemsg.Kind, erasemsg.Data)

	log.Printf("Hash: %x\n", hash)

	chunks, err = MessageFirmwareUpload(payload, hash)
	if err != nil {
		return err
	}
	uploadmsg, err := sendToDevice(dev, chunks)
	if err != nil {
		return err
	}
	log.Printf("Success %d! FirmwareUpload %s\n", uploadmsg.Kind, uploadmsg.Data)

	// Send ButtonAck
	chunks, err = MessageButtonAck()
	if err != nil {
		return err
	}
	return sendToDeviceNoAnswer(dev, chunks)
}

// GetFeatures send Features message to the device
func (d *Device) GetFeatures() (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	chunks, err := MessageGetFeatures()
	if err != nil {
		return wire.Message{}, err
	}

	return sendToDevice(dev, chunks)
}

// GenerateMnemonic Ask the device to generate a mnemonic and configure itself with it.
func (d *Device) GenerateMnemonic(wordCount uint32, usePassphrase bool) (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	chunks, err := MessageGenerateMnemonic(wordCount, usePassphrase)
	if err != nil {
		return wire.Message{}, err
	}
	msg, err := sendToDevice(dev, chunks)
	if err != nil {
		return wire.Message{}, err
	}

	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = deviceButtonAck(dev)
		if err != nil {
			return wire.Message{}, err
		}
	}

	return msg, err
}

// RecoveryDevice ask the device to perform the seed backup
func (d *Device) Recovery(wordCount uint32, usePassphrase, dryRun bool) (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()
	var msg wire.Message
	var chunks [][64]byte

	log.Printf("Using passphrase %t\n", usePassphrase)
	chunks, err = MessageRecovery(wordCount, usePassphrase, dryRun)
	if err != nil {
		return wire.Message{}, err
	}
	msg, err = sendToDevice(dev, chunks)
	if err != nil {
		return msg, err
	}
	log.Printf("Recovery device %d! Answer is: %s\n", msg.Kind, msg.Data)

	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = deviceButtonAck(dev)
		if err != nil {
			return wire.Message{}, err
		}
	}

	return msg, nil
}

// SetMnemonic Configure the device with a mnemonic.
func (d *Device) SetMnemonic(mnemonic string) (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	// Send SetMnemonic
	chunks, err := MessageSetMnemonic(mnemonic)
	if err != nil {
		return wire.Message{}, err
	}
	msg, err := sendToDevice(dev, chunks)
	if err != nil {
		return wire.Message{}, err
	}

	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = deviceButtonAck(dev)
		if err != nil {
			return wire.Message{}, err
		}
	}

	return msg, err
}

// SignMessage Ask the device to sign a message using the secret key at given index.
func (d *Device) SignMessage(addressN int, message string) (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	chunks, err := MessageSignMessage(addressN, message)
	if err != nil {
		return wire.Message{}, err
	}
	return sendToDevice(dev, chunks)
}

// TransactionSign Ask the device to sign a transaction using the given information.
func (d *Device) TransactionSign(inputs []*messages.SkycoinTransactionInput, outputs []*messages.SkycoinTransactionOutput) (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	chunks, err := MessageTransactionSign(inputs, outputs)
	if err != nil {
		return wire.Message{}, err
	}
	return sendToDevice(dev, chunks)
}

// WipeDevice wipes out device configuration
func (d *Device) Wipe() (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}

	defer dev.Close()
	var chunks [][64]byte

	err = initialize(dev)
	if err != nil {
		return wire.Message{}, err
	}

	chunks, err = MessageWipe()
	if err != nil {
		return wire.Message{}, err
	}

	var msg wire.Message
	msg, err = sendToDevice(dev, chunks)
	if err != nil {
		return wire.Message{}, err
	}
	log.Printf("Wipe device %d! Answer is: %x\n", msg.Kind, msg.Data)

	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		msg, err = deviceButtonAck(dev)
		if err != nil {
			return wire.Message{}, err
		}
	}

	if msg.Kind == uint16(messages.MessageType_MessageType_ButtonRequest) {
		err = initialize(dev)
		if err != nil {
			return wire.Message{}, err
		}
	}

	return msg, err
}

// ButtonAck when the device is waiting for the user to press a button
// the PC need to acknowledge, showing it knows we are waiting for a user action
func (d *Device) ButtonAck() (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()
	return deviceButtonAck(dev)
}

func deviceButtonAck(dev io.ReadWriteCloser) (wire.Message, error) {
	var msg wire.Message
	// Send ButtonAck
	chunks, err := MessageButtonAck()
	if err != nil {
		return msg, err
	}
	err = sendToDeviceNoAnswer(dev, chunks)
	if err != nil {
		return msg, err
	}

	_, err = msg.ReadFrom(dev)
	time.Sleep(1 * time.Second)
	if err != nil {
		return msg, err
	}
	return msg, nil
}

// PassphraseAck send this message when the device is waiting for the user to input a passphrase
func (d *Device) PassphraseAck(passphrase string) (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	chunks, err := MessagePassphraseAck(passphrase)
	if err != nil {
		return wire.Message{}, err
	}
	return sendToDevice(dev, chunks)
}

// WordAck send a word to the device during device "recovery procedure"
func (d *Device) WordAck(word string) (wire.Message, error) {
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}

	defer dev.Close()
	chunks, err := MessageWordAck(word)
	if err != nil {
		return wire.Message{}, err
	}
	msg, err := sendToDevice(dev, chunks)
	if err != nil {
		return wire.Message{}, err
	}

	return msg, nil
}

// PinMatrixAck during PIN code setting use this message to send user input to device
func (d *Device) PinMatrixAck(p string) (wire.Message, error) {
	time.Sleep(1 * time.Second)
	dev, err := d.GetDevice()
	if err != nil {
		return wire.Message{}, err
	}
	defer dev.Close()

	log.Printf("Setting pin: %s\n", p)

	chunks, err := MessagePinMatrixAck(p)
	if err != nil {
		return wire.Message{}, nil
	}
	return sendToDevice(dev, chunks)
}

// GetDevice is a wrapper func around getDevice
func (d *Device) GetDevice() (io.ReadWriteCloser, error) {
	return getDevice(d.DeviceType)
}
