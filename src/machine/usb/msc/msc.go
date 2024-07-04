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

var MscInstance *msc

type msc struct {
	buf       *RingBuffer
	rxHandler func([]byte)
	txHandler func()
	waitTxc   bool
}

func init() {
	if MscInstance == nil {
		MscInstance = newMsc()
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

	return m
}

func Port() *msc {
	return MscInstance
}

// SetRxHandler sets the handler function for incoming messages.
func (m *msc) SetRxHandler(rxHandler func([]byte)) {
	m.rxHandler = rxHandler
}

// SetTxHandler sets the handler function for outgoing messages.
func (m *msc) SetTxHandler(txHandler func()) {
	m.txHandler = txHandler
}

/*func (m *msc) Write(b []byte) (n int, err error) {
	s, e := 0, 0
	for s = 0; s < len(b); s += 4 {
		e = s + 4
		if e > len(b) {
			e = len(b)
		}

		m.tx(b[s:e])
	}
	return e, nil
}*/

// sendUSBPacket sends a MSC Packet.
func (m *msc) sendUSBPacket(b []byte) {
	machine.SendUSBInPacket(usb.MSC_ENDPOINT_IN, b)
}

// from BulkIn
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

func (m *msc) tx(b []byte) {
	if machine.USBDev.InitEndpointComplete {
		if m.waitTxc {
			m.buf.Put(b)
		} else {
			m.waitTxc = true
			m.sendUSBPacket(b)
		}
	}
}

// from BulkOut
func (m *msc) RxHandler(b []byte) {
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
