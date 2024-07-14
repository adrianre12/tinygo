package msc

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

var (
	recvChan     chan []byte
	stopChan     chan interface{}
	recvChanWg   sync.WaitGroup
	currentState = StateCommand
)

const (
	StateCommand = iota
	StateSend
	StateRecv
	StateStatus
)

func initStateMachine() {
	if recvChan != nil {
		close(stopChan)
		recvChanWg.Wait()
	}
	recvChan = make(chan []byte, 9)
	stopChan = make(chan interface{})

	recvChanWg.Add(1)
	go botRoutine()

}

func botRoutine() {
	var buf []byte
	for {
		select {
		case <-stopChan:
			return
		case buf = <-recvChan:
			botSM(buf)
		}
	}
}

func botSM(buf []byte) {
	switch currentState {
	case StateCommand:
		fmt.Printf("Cmd: buf=% x\n", buf)
		cbw := CBW{
			DCBWSignature:          binary.LittleEndian.Uint32(buf[0:4]),
			DCBWTag:                binary.LittleEndian.Uint32(buf[4:8]),
			DCBWDataTransferLength: binary.LittleEndian.Uint32(buf[8:12]),
			BmCBWFlags:             buf[12],
			BCBWLUN:                buf[13],
			BCBWCBLength:           buf[14],
			CBWCB:                  buf[15:31],
		}
		scsiCommands(cbw)
	case StateSend:

	case StateRecv:

	case StateStatus:
	}
}

type CBW struct {
	DCBWSignature          uint32
	DCBWTag                uint32
	DCBWDataTransferLength uint32
	BmCBWFlags             uint8
	BCBWLUN                uint8
	BCBWCBLength           uint8
	CBWCB                  []byte
}

type CBS struct {
	DCBWSignature   uint32
	DCBWTag         uint32
	DCBWDataResidue uint32
	BmCBWStatus     uint8
}

func (s *CBS) ToBytes() []byte {
	b := make([]byte, 13)
	binary.LittleEndian.PutUint32(b[0:], s.DCBWSignature)
	binary.LittleEndian.PutUint32(b[4:], s.DCBWTag)
	binary.LittleEndian.PutUint32(b[8:], s.DCBWDataResidue)
	b[12] = s.BmCBWStatus

	return b
}

const (
	scsiInquiry        = 0x12
	scsiRequestSense   = 0x03
	scsiModeSense      = 0x1A
	scsiTestUnitReady  = 0x00
	scsiReadCapacity10 = 0x25
	scsiRead10         = 0x28
	scsiWrite10        = 0x2A

	cswStatusPass       = 0x00
	cswStatusFail       = 0x01
	cswStatusPhaseError = 0x02

	senseIlegalRequest     = 0x05
	senseInvalidComandASC  = 0x20
	senseInvalidComandASCQ = 0x00
)

func scsiCommands(cbw CBW) {
	fmt.Printf("scsi Tag=%d CBWCB % x\n", cbw.DCBWTag, cbw.CBWCB)
	cbs := CBS{
		DCBWSignature:   0x53425355,
		DCBWTag:         cbw.DCBWTag,
		DCBWDataResidue: 0,
		BmCBWStatus:     cswStatusFail,
	}

	switch cbw.CBWCB[0] {
	case scsiInquiry:
		fmt.Println("Inquiery")

		cbs.BmCBWStatus = cmdInquiry(cbw.CBWCB)

	case scsiRequestSense:
		fmt.Println("RequestSense")

		cbs.BmCBWStatus = cmdRequestSense(cbw.CBWCB)

	case scsiTestUnitReady:
		fmt.Println("TestUnitReady")

		cbs.BmCBWStatus = cmdTestUnitReady(cbw.CBWCB)

	case scsiReadCapacity10:
		fmt.Println("ReadCapacity10")

		cbs.BmCBWStatus = cmdReadCapacity10(cbw.CBWCB)

	case scsiModeSense:
		fmt.Println("ModeSense")

		cbs.BmCBWStatus = cmdModeSense(cbw.CBWCB)

	case scsiRead10:
		fmt.Println("Read10")

		cbs.BmCBWStatus = cmdRead10(cbw.CBWCB)

	default:
		fmt.Printf("Unknown SCSI cmd 0x%X\n", cbw.CBWCB[0])
		senseKey = senseIlegalRequest
		senseCode = senseInvalidComandASC
		senseCodeQualifier = senseInvalidComandASCQ
		cbs.BmCBWStatus = cswStatusFail
	}

	fmt.Printf("cbs % x\n", cbs.ToBytes())
	currentState = StateStatus
	Port().Tx(cbs.ToBytes())
	currentState = StateCommand

}

type InquiryResponse struct {
	PQPT            uint8
	RMB             uint8
	Version         uint8
	Flags3          uint8
	AditionalLength uint8
	Flags5          uint8
	Flags6          uint8
	Flags7          uint8
	VendorId        [8]byte
	ProductId       [16]byte
	ProductRevision [4]byte
}

func (s *InquiryResponse) ToBytes() []byte {
	b := make([]byte, 36)
	b[0] = s.PQPT
	b[1] = s.RMB
	b[2] = s.Version
	b[3] = s.Flags3
	b[4] = s.AditionalLength
	b[5] = s.Flags5
	b[6] = s.Flags6
	b[7] = s.Flags7
	copy(b[8:], s.VendorId[:])
	copy(b[16:], s.ProductId[:])
	copy(b[32:], s.ProductRevision[:])

	return b
}

func cmdInquiry(cb []byte) uint8 {
	currentState = StateSend
	response := InquiryResponse{
		PQPT:            0,
		RMB:             0x80,
		Version:         0x04,
		Flags3:          0x02,
		AditionalLength: 0x1F,
		Flags5:          0,
		Flags6:          0,
		Flags7:          0,
	}
	copy(response.VendorId[:], "Vendor  ") //shorter is probably ok
	copy(response.ProductId[:], "Identification  ")
	copy(response.ProductRevision[:], "0002")

	Port().Tx(response.ToBytes())

	return cswStatusPass
}

type RequestSenseResponse struct {
	ErrorCode                    uint8
	SegmentNumber                uint8
	Sensekey                     uint8
	Information                  uint32
	AditionalSenseLength         uint8
	CommandSpecificInformation   uint32
	AdditionalSenceCode          uint8
	AdditionalSenceCodeQualifier uint8
	FieldReplaceableUnitCode     uint8
	Flags15                      uint8
	FieldPointer                 uint16
}

func (s *RequestSenseResponse) ToBytes() []byte {
	b := make([]byte, 18)
	b[0] = s.ErrorCode
	b[1] = s.SegmentNumber
	b[2] = s.Sensekey
	binary.BigEndian.PutUint32(b[3:], s.Information)
	b[7] = s.AditionalSenseLength
	binary.BigEndian.PutUint32(b[8:], s.CommandSpecificInformation)
	b[12] = s.AdditionalSenceCode
	b[13] = s.AdditionalSenceCodeQualifier
	b[14] = s.FieldReplaceableUnitCode
	b[15] = s.Flags15
	binary.BigEndian.PutUint16(b[16:], s.FieldPointer)

	return b
}

var (
	senseKey           uint8
	senseCode          uint8
	senseCodeQualifier uint8
)

func cmdRequestSense(cb []byte) uint8 {
	currentState = StateSend
	response := RequestSenseResponse{
		ErrorCode:                    0x70,
		SegmentNumber:                0x00,
		Sensekey:                     senseKey,
		Information:                  0x00000000,
		AditionalSenseLength:         0x0A,
		CommandSpecificInformation:   0x00000000,
		AdditionalSenceCode:          senseCode,
		AdditionalSenceCodeQualifier: senseCodeQualifier,
		FieldReplaceableUnitCode:     0x00,
		Flags15:                      0x00,
		FieldPointer:                 0x00,
	}

	Port().Tx(response.ToBytes())

	return cswStatusPass
}

func cmdTestUnitReady(cb []byte) uint8 {

	// do some checks to see if it is ready

	return cswStatusPass
}

type ReadCapacity10Respose struct {
	ReturnedLogicalBlockAddress uint32
	BlockLengthInBytes          uint32
}

func (s *ReadCapacity10Respose) ToBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b[0:], s.ReturnedLogicalBlockAddress)
	binary.BigEndian.PutUint32(b[4:], s.BlockLengthInBytes)

	return b
}

func cmdReadCapacity10(cb []byte) uint8 {
	currentState = StateSend
	response := ReadCapacity10Respose{
		ReturnedLogicalBlockAddress: MaxLogicalBlocks,
		BlockLengthInBytes:          BlockSize,
	}

	Port().Tx(response.ToBytes())

	return cswStatusPass
}

func cmdModeSense(cb []byte) uint8 {
	currentState = StateSend
	response := []byte{ // Hacky but this just says write protect off, no caching and no other data pages.
		0x03,
		0x00,
		0x00,
		0x00,
	}
	Port().Tx(response)

	return cswStatusPass
}

func cmdRead10(cb []byte) uint8 {
	currentState = StateSend
	//flags1 := cb[1]
	lba := binary.BigEndian.Uint32(cb[2:])
	//grpNo := cb[6]
	transLen := binary.BigEndian.Uint16(cb[7:])
	//control := b[9]
	fmt.Printf("lba=%d translen=%d\n", lba, transLen)
	for i := range transLen * 8 {
		response := make([]byte, 64)
		Port().Tx(response)
		fmt.Printf("blockNum %d\n", i)
		time.Sleep(10 * time.Millisecond)
	}
	return cswStatusPass
}
