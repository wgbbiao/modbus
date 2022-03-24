package main

import (
	"encoding/binary"
)

const (
	rtuAduMinSize = 4 // address(1) + funcCode(1) + crc(2)

)

// uint162Bytes creates a sequence of uint16 data.
func uint162Bytes(value ...uint16) []byte {
	data := make([]byte, 2*len(value))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}

// for CRC-16/MODBUS
func crc16(data []byte) []byte {
	var crc16 uint16 = 0xFFFF
	for _, v := range data {
		crc16 ^= uint16(v)
		for i := 0; i < 8; i++ {
			if crc16&0x0001 == 1 {
				crc16 = (crc16 >> 1) ^ 0xA001
			} else {
				crc16 >>= 1
			}
		}
	}
	data = append(data, byte(crc16&0xFF))
	data = append(data, byte(crc16>>8&0xFF))
	return data
}

func calculateResponseLength(adu []byte) int {
	length := rtuAduMinSize
	switch adu[1] {
	case FuncCodeReadDiscreteInputs,
		FuncCodeReadCoils:
		count := int(binary.BigEndian.Uint16(adu[4:]))
		length += 1 + count/8
		if count%8 != 0 {
			length++
		}
	case FuncCodeReadInputRegisters,
		FuncCodeReadHoldingRegisters:
		count := int(binary.BigEndian.Uint16(adu[4:]))
		length += 1 + count*2
	case FuncCodeWriteSingleCoil,
		FuncCodeWriteMultipleCoils,
		FuncCodeWriteSingleRegister,
		FuncCodeWriteMultipleRegisters:
		length += 4
	default:
	}
	return length
}
