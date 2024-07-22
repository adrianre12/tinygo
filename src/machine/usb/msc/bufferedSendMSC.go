package msc

import (
	"fmt"
	"machine"
	"machine/usb"
	"runtime/volatile"
)

const (
	bufferSize  = 16
	pageSize    = 64
	isNotActive = 0
	isActive    = 1
)

type BufferedSendMSC struct {
	buffer     [bufferSize][]byte
	head       volatile.Register8
	tail       volatile.Register8
	activeFlag volatile.Register8
	fullSignal chan struct{}
}

var Led machine.Pin

// NewBufferedSendMSC returns a new send buffer.
func NewBufferedSendMSC() *BufferedSendMSC {
	buf := BufferedSendMSC{}
	for i := range buf.buffer {
		buf.buffer[i] = make([]byte, 0, pageSize)
	}
	buf.fullSignal = make(chan struct{})

	Led = machine.LED
	Led.Configure(machine.PinConfig{
		Mode: machine.PinOutput,
	})
	Led.Low()

	return &buf
}

func (m *BufferedSendMSC) sendFullSignal() {
	select {
	case m.fullSignal <- struct{}{}:
		{
		}
	default:
		{
		}
	}
}

// Put stores a byte in the buffer. If the buffer is already
// full, the method will return false.
func (m *BufferedSendMSC) put(val []byte) {
	nextHead := (m.head.Get() + 1) % bufferSize
	for m.tail.Get() == nextHead {
		fmt.Println("waiting")
		<-m.fullSignal
	}
	m.buffer[nextHead] = m.buffer[nextHead][:len(val)] //set the size large enough to copy the val into
	copy(m.buffer[nextHead][:], val)
	m.head.Set(nextHead)
}

// Get returns a []byte from the buffer. If the buffer is empty,
// the method will return a false as the second value.
func (m *BufferedSendMSC) get() ([]byte, bool) {
	if m.head.Get() == m.tail.Get() {
		return nil, false
	}

	m.tail.Set((m.tail.Get() + 1) % bufferSize)
	return m.buffer[m.tail.Get()][:], true
}

// Clear resets the head and length to zero.
func (m *BufferedSendMSC) Clear() {
	m.head.Set(0)
	m.tail.Set(0)
}

// Bulk In Interrupt
func (m *BufferedSendMSC) TxHandler() {
	if b, ok := m.get(); ok {
		m.sendUSBInPacket(b)
	} else {
		m.activeFlag.Set(isNotActive)
	}
	m.sendFullSignal()
}

// sendUSBPacket sends a MSC Packet.
func (m *BufferedSendMSC) sendUSBInPacket(b []byte) {
	m.activeFlag.Set(isActive)
	machine.SendUSBInPacket(usb.MSC_ENDPOINT_IN, b)
}

// bufSendUSBPacket buffers or sends a MSC Packet.
func (m *BufferedSendMSC) sendUSBPacket(b []byte) {
	fmt.Printf("Packet size %d\n", len(b))

	/*if m.activeFlag.Get() == isActive {
		fmt.Printf("Buffer Packet %v\n", b)
		m.put(b)
	} else {
		fmt.Printf("Send Packet %v\n", b)

		m.sendUSBInPacket(b)
	}*/

	fmt.Printf("Buffer Packet %v\n", b)
	m.put(b)

	if m.activeFlag.Get() == isNotActive {
		m.TxHandler()
	}

}

func (m *BufferedSendMSC) SendUSB(b []byte) {
	if !machine.USBDev.InitEndpointComplete {
		return
	}
	count := len(b)
	numPackets := count / pageSize
	if len(b)%pageSize > 0 || count == 0 {
		numPackets++
	}
	fmt.Printf("count=%d, numPackets=%d\n", count, numPackets)
	var start int
	var end int
	for p := range numPackets {
		start = p * pageSize
		end = start + pageSize
		if end > count {
			end = count
		}
		m.sendUSBPacket(b[start:end])
		//time.Sleep(30 * time.Millisecond)
	}

}
