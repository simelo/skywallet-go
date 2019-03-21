package devicewallet

import (
	"bytes"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	messages "github.com/skycoin/hardware-wallet-go/src/device-wallet/messages/go"
	"github.com/skycoin/hardware-wallet-go/src/device-wallet/wire"
)

type devicerSuit struct {
	suite.Suite
}

func (suite *devicerSuit) SetupTest() {
}

func TestDevicerSuitSuit(t *testing.T) {
	suite.Run(t, new(devicerSuit))
}

type testHelperCloseableBuffer struct {
	bytes.Buffer
}

func (cwr testHelperCloseableBuffer) Read(p []byte) (n int, err error) {
	return 0, nil
}
func (cwr testHelperCloseableBuffer) Write(p []byte) (n int, err error) {
	return 0, nil
}
func (cwr testHelperCloseableBuffer) Close() error {
	return nil
}

func (suite *devicerSuit) TestGenerateMnemonic() {
	// NOTE(denisacostaq@gmail.com): Giving
	driverMock := &MockDeviceDriver{}
	driverMock.On("GetDevice").Return(&testHelperCloseableBuffer{}, nil)
	driverMock.On("SendToDevice", mock.Anything, mock.Anything).Return(
		wire.Message{Kind: uint16(messages.MessageType_MessageType_EntropyRequest), Data: nil}, nil)
	device := Device{driverMock}

	// NOTE(denisacostaq@gmail.com): When
	msg, err := device.GenerateMnemonic(12, false)

	// NOTE(denisacostaq@gmail.com): Assert
	suite.Nil(err)
	driverMock.AssertCalled(suite.T(), "GetDevice")
	driverMock.AssertNumberOfCalls(suite.T(), "SendToDevice", 2)
	mock.AssertExpectationsForObjects(suite.T(), driverMock)
	spew.Dump(msg)
}
