package msc

import (
	"fmt"
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
	buf       *RingBufferMSC
	rxHandler func([]byte)
	txHandler func()
	waitTxc   bool
}

func init() {
	if mscInstance == nil {
		mscInstance = newMsc()
	}
}

func newMsc() *msc {
	m := &msc{
		buf: NewRingBuffer(),
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
				TxHandler: m.TxHandler,
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

func Port() *msc {
	return mscInstance
}

// SetRxHandler sets the handler function for incoming messages.
func (m *msc) SetRxHandler(rxHandler func([]byte)) {
	m.rxHandler = rxHandler
}

// SetTxHandler sets the handler function for outgoing messages.
func (m *msc) SetTxHandler(txHandler func()) {
	m.txHandler = txHandler
}

// sendUSBPacket sends a MSC Packet.
func (m *msc) sendUSBPacket(b []byte) {
	machine.SendUSBInPacket(usb.MSC_ENDPOINT_IN, b)
}

// BulkIn
func (m *msc) TxHandler() {
	if m.txHandler != nil {
		m.txHandler()
	}

	m.waitTxc = false
	if b, ok := m.buf.Get(); ok {
		m.waitTxc = true
		m.sendUSBPacket(b)
	}
}

func (m *msc) Tx(b []byte) {
	if machine.USBDev.InitEndpointComplete {
		if m.waitTxc {
			fmt.Println("Putting packet")

			m.buf.Put(b)
			fmt.Printf("Used %d\n", m.buf.Used())
		} else {
			m.waitTxc = true
			m.sendUSBPacket(b)
			fmt.Println("Sent packet")
		}
	}
}

// from BulkOut
func (m *msc) RxHandler(b []byte) {
	recvChan <- b
	if m.rxHandler != nil {
		m.rxHandler(b)
	}
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
