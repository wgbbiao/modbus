package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tarm/serial"
)

const (
	FuncCodeReadCoils              = 1
	FuncCodeReadDiscreteInputs     = 2
	FuncCodeReadHoldingRegisters   = 3
	FuncCodeReadInputRegisters     = 4
	FuncCodeWriteSingleCoil        = 5
	FuncCodeWriteSingleRegister    = 6
	FuncCodeWriteMultipleCoils     = 15
	FuncCodeWriteMultipleRegisters = 16
)

type Client struct {
	serialPort       *serial.Port
	serialPortConfig *serial.Config
	serialPortName   string
	showLog          bool

	BaudRate int
}

func NewClient(c *serial.Config) (*Client, error) {
	// c := &serial.Config{Name: "COM3", Baud: 9600, StopBits: 1, Parity: serial.ParityNone}
	s, err := serial.OpenPort(c)
	cc := &Client{serialPort: s, serialPortConfig: c, serialPortName: c.Name, BaudRate: c.Baud}
	cc.showLog = false
	return cc, err
}

//发送
func (c *Client) Send(data []byte) ([]byte, error) {
	if c.showLog {
		log.Printf("ReadCoils: 发送[% x]", data)
	}
	// 清空缓冲区
	c.serialPort.Flush()
	n, err := c.serialPort.Write(data)
	if err != nil {
		return nil, err
	}
	if n != len(data) {
		return nil, fmt.Errorf("发送长度不一致")
	}
	bytesToRead := calculateResponseLength(data)

	time.Sleep(c.calculateDelay(len(data) + bytesToRead))

	buf := make([]byte, 64)
	n, err = c.serialPort.Read(buf)

	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("读取超时")
	}
	// 验证数据

	return buf[:n], nil
}

func (c *Client) Close() {
	c.serialPort.Close()
}

func (sf *Client) calculateDelay(chars int) time.Duration {
	var characterDelay, frameDelay int // us

	if sf.BaudRate <= 0 || sf.BaudRate > 19200 {
		characterDelay = 750
		frameDelay = 1750
	} else {
		characterDelay = 15000000 / sf.BaudRate
		frameDelay = 35000000 / sf.BaudRate
	}
	return time.Duration(characterDelay*chars+frameDelay) * time.Microsecond * 2
}

// 开启log
func (c *Client) EnableLog() {
	c.showLog = true
}

// WriteSingleCoil write a single output to either ON or OFF in a
// remote device and returns success or failed.
func (c *Client) WriteSingleCoil(slaveID byte, address uint16, isOn bool) error {
	return nil
}

// WriteSingleRegister writes a single holding register in a remote
// device and returns success or failed.
func (c *Client) WriteSingleRegister(slaveID byte, address, value uint16) error {
	return nil
}

// ReadHoldingRegisters reads the contents of a contiguous block of
// holding registers in a remote device and returns register value.
func (c *Client) ReadHoldingRegisters(slaveID byte, address, quantity uint16) (results []uint16, err error) {
	return nil, nil
}

// ReadCoils reads from 1 to 2000 contiguous status of coils in a
// remote device and returns coil status.
func (c *Client) ReadCoils(slaveID byte, address, quantity uint16) (results []byte, err error) {
	data := []byte{slaveID, FuncCodeReadCoils}
	data = append(data, uint162Bytes(address, quantity)...)
	data = crc16(data)
	res, err := c.Send(data)

	if c.showLog {
		log.Printf("ReadCoils: 接收[% x]", res)
	}
	//取出数据
	if err != nil {
		return nil, err
	}
	if len(res) < 4 {
		return nil, fmt.Errorf("数据长度不够")
	}
	if res[0] != slaveID {
		return nil, fmt.Errorf("从机ID不一致")
	}
	if res[1] != FuncCodeReadCoils {
		return nil, fmt.Errorf("功能码不一致")
	}
	return res[3 : len(res)-2], err
}

// ReadDiscreteInputs reads from 1 to 2000 contiguous status of
// discrete inputs in a remote device and returns input status.
func (c *Client) ReadDiscreteInputs(slaveID byte, address, quantity uint16) (results []byte, err error) {
	return nil, nil
}