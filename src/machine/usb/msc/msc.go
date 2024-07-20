package msc

import (
	"machine"
	"machine/usb"
	"machine/usb/descriptor"
)

const (
	// bmRequestType
	usb_REQUEST_HOSTTODEVICE                 = 0x00
	usb_REQUEST_CLASS                        = 0x20
	usb_REQUEST_INTERFACE                    = 0x01
	usb_REQUEST_HOSTTODEVICE_CLASS_INTERFACE = (usb_REQUEST_HOSTTODEVICE | usb_REQUEST_CLASS | usb_REQUEST_INTERFACE)

	// MSC Class Request
	usb_BOM_Storage_Reset = 0xFF
	usb_GetMaxLun         = 0xFE
)

var (
	MaxLogicalBlocks uint32 = 0x00000800
	BlockSize        uint32 = 0x00000200
)

var mscInstance *msc

type msc struct {
	buf *BufferedSendMSC
}

func init() {
	if mscInstance == nil {
		mscInstance = newMsc()
	}
}

func newMsc() *msc {
	m := &msc{
		buf: NewBufferedSendMSC(),
	}
	machine.ConfigureUSBEndpoint(descriptor.MSC,
		[]usb.EndpointConfig{
			{
				Index:     usb.MSC_ENDPOINT_OUT,
				IsIn:      false,
				Type:      usb.ENDPOINT_TYPE_BULK,
				RxHandler: m.RxHandler,
			},
			{
				Index:     usb.MSC_ENDPOINT_IN,
				IsIn:      true,
				Type:      usb.ENDPOINT_TYPE_BULK,
				TxHandler: m.buf.TxHandler,
			},
		},
		[]usb.SetupConfig{
			{
				Index:   2,
				Handler: mscSetup,
			},
		})

	initStateMachine()

	return m
}

func mscSetup(setup usb.Setup) bool {

	if setup.BmRequestType == usb_REQUEST_HOSTTODEVICE_CLASS_INTERFACE {
		//Bulk-Only Mass Storage Reset
		if setup.BRequest == usb_BOM_Storage_Reset {
			machine.SendZlp()
			return true
		}

		//Get Max Lun
		if setup.BRequest == usb_GetMaxLun {
			machine.SendUSBInPacket(0, []byte{0})
			return true
		}
	}

	return false
}

func Port() *msc {
	return mscInstance
}

func (m *msc) Clear() {
	m.buf.Clear()
}

func (m *msc) Tx(b []byte) {
	if machine.USBDev.InitEndpointComplete {

		//split to packets
		m.buf.SendUSBPacket(b)

	}
}

// from BulkOut
func (m *msc) RxHandler(b []byte) {
	recvChan <- b
}
