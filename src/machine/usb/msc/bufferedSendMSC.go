package msc

import (
	"fmt"
	"machine"
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
	length     volatile.Register8
	tmp        uint8 // tmp var to avoid allocations during interrupts
	activeFlag volatile.Register8
	fullSignal chan struct{}
}

// NewBufferedSendMSC returns a new send buffer.
func NewBufferedSendMSC() *BufferedSendMSC {
	buf := BufferedSendMSC{}
	for i := range buf.buffer {
		buf.buffer[i] = make([]byte, 0, pageSize)
	}
	buf.fullSignal = make(chan struct{})
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
	for m.length.Get() == bufferSize {
		fmt.Println("waiting")
		<-m.fullSignal
	}
	m.tmp = m.head.Get()
	m.buffer[m.tmp] = m.buffer[m.tmp][:len(val)]
	copy(m.buffer[m.tmp][:], val)
	m.head.Set((m.head.Get() + 1) % bufferSize)
	m.length.Set(m.length.Get() + 1)
}

// Get returns a []byte from the buffer. If the buffer is empty,
// the method will return a false as the second value.
func (m *BufferedSendMSC) get() ([]byte, bool) {
	if m.length.Get() > 0 {
		m.tmp = (m.head.Get() - m.length.Get()) % bufferSize //find the tail
		m.length.Set(m.length.Get() - 1)
		return m.buffer[m.tmp][:], true
	}

	return nil, false
}

// Clear resets the head and length to zero.
func (m *BufferedSendMSC) Clear() {
	m.head.Set(0)
	m.length.Set(0)
}

// Bulk In
func (m *BufferedSendMSC) TxHandler() {
	fmt.Println("TxHandler")

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
	machine.SendUSBInPacket(0, b)
}

// bufSendUSBPacket buffers or sends a MSC Packet.
func (m *BufferedSendMSC) SendUSBPacket(b []byte) {
	if m.activeFlag.Get() == isActive {
		fmt.Printf("Buffer Packet %v\n", b)
		m.put(b)
	} else {
		m.sendUSBInPacket(b)
	}
}
