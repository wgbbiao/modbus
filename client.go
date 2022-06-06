package modbus

import (
	"fmt"
	"log"
	"sync"
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

// proto address limit.
const (
	AddressBroadCast = 0
	AddressMin       = 1
	AddressMax       = 247

	// Bits
	WriteRegQuantityMin = 1   // 1
	WriteRegQuantityMax = 123 // 0x007b
)

// ProtocolDataUnit (PDU) is independent of underlying communication layers.
type ProtocolDataUnit struct {
	FuncCode byte
	Data     []byte
}

type Client struct {
	serialPort       *serial.Port
	serialPortConfig *serial.Config
	serialPortName   string
	showLog          bool

	//发送前延迟 默认 100ms
	DelayRtsBeforeSend time.Duration

	BaudRate int
	// 同步锁
	ml sync.Mutex

	addressMin byte
	addressMax byte

	// 延时倍数
	DelayTimes int
}

func NewClient(c *serial.Config) (*Client, error) {
	s, err := serial.OpenPort(c)
	cc := &Client{
		serialPort:       s,
		serialPortConfig: c,
		serialPortName:   c.Name,
		BaudRate:         c.Baud,
		addressMax:       AddressMax,
		addressMin:       AddressMin,
		DelayTimes:       2,
	}
	if cc.DelayRtsBeforeSend == 0 {
		cc.DelayRtsBeforeSend = time.Millisecond * 100
	}
	cc.showLog = false
	return cc, err
}

//发送
func (c *Client) Send(data []byte) ([]byte, error) {
	// 清空缓冲区
	c.ml.Lock()
	defer c.ml.Unlock()
	time.Sleep(c.DelayRtsBeforeSend)
	if c.serialPort == nil {
		return nil, fmt.Errorf("serialPort is nil")
	}
	if err := c.serialPort.Flush(); err != nil {
		return nil, err
	}
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
	if c.serialPort != nil {
		c.serialPort.Close()
	}
}

// 重新连接
func (c *Client) Reconnect() error {
	if c.serialPort != nil {
		c.serialPort.Close()
	}
	s, err := serial.OpenPort(c.serialPortConfig)
	if err != nil {
		return err
	}
	c.serialPort = s
	return nil
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
	return time.Duration(characterDelay*chars+frameDelay) * time.Microsecond * time.Duration(sf.DelayTimes)
}

// 开启log
func (c *Client) EnableLog() {
	c.showLog = true
}

// WriteSingleCoil 在远程设备中将单个输出写入ON或OFF，并返回成功或失败。
func (c *Client) WriteSingleCoil(slaveID byte, address uint16, isOn bool) error {
	var value uint16
	if isOn { // The requested ON/OFF state can only be 0xFF00 and 0x0000
		value = 0xFF00
	}
	data := []byte{slaveID, FuncCodeWriteSingleCoil}
	data = append(data, uint162Bytes(address, value)...)
	data = crc16(data)
	c.Printf("WriteSingleCoil: 发送[% x]", data)
	res, err := c.Send(data)
	c.Printf("WriteSingleCoil: 接收[% x]", res)

	if err != nil {
		return err
	}
	return nil
}

// WriteSingleRegister 在远程设备中写入单个保留寄存器，并返回成功或失败。
func (c *Client) WriteSingleRegister(slaveID byte, address, value uint16) error {
	data := []byte{slaveID, FuncCodeWriteSingleRegister}
	data = append(data, uint162Bytes(address, value)...)
	data = crc16(data)
	c.Printf("WriteSingleRegister: 发送[% x]", data)
	res, err := c.Send(data)
	c.Printf("WriteSingleRegister: 接收[% x]", res)
	return err
}

// ReadHoldingRegisters 读取远程设备中连续的保持寄存器块的内容，并返回寄存器值。
func (c *Client) ReadHoldingRegisters(slaveID byte, address, quantity uint16) (results []uint16, err error) {
	data := []byte{slaveID, FuncCodeReadHoldingRegisters}
	data = append(data, uint162Bytes(address, quantity)...)
	data = crc16(data)
	c.Printf("ReadHoldingRegisters: 发送[% x]", data)
	res, err := c.Send(data)
	c.Printf("ReadHoldingRegisters: 接收[% x]", res)

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
	if res[1] != FuncCodeReadHoldingRegisters {
		return nil, fmt.Errorf("功能码不一致")
	}
	return bytes2Uint16(res[3 : len(res)-2]), err
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x10)
//  Starting address      : 2 bytes
//  Quantity of outputs   : 2 bytes
//  Byte count            : 1 byte
//  Registers value       : N* bytes
// Response:
//  Function code         : 1 byte (0x10)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
func (c *Client) WriteMultipleRegistersBytes(slaveID byte, address, quantity uint16, value []byte) error {
	if slaveID > c.addressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, c.addressMax)
	}
	if quantity < WriteRegQuantityMin || quantity > WriteRegQuantityMax {
		return fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, WriteRegQuantityMin, WriteRegQuantityMax)
	}

	if len(value) != int(quantity*2) {
		return fmt.Errorf("modbus: value length '%v' does not twice as quantity '%v'", len(value), quantity)
	}
	data := []byte{slaveID, FuncCodeWriteMultipleRegisters}
	data = append(data, pduDataBlockSuffix(value, address, quantity)...)
	data = crc16(data)

	c.Printf("WriteMultipleRegistersBytes: 发送[% x]", data)
	res, err := c.Send(data)
	c.Printf("WriteMultipleRegistersBytes: 接收[% x]", res)
	//取出数据
	if err != nil {
		return err
	}
	if len(res) < 4 {
		return fmt.Errorf("数据长度不够")
	}
	if res[0] != slaveID {
		return fmt.Errorf("从机ID不一致")
	}
	if res[1] != FuncCodeWriteMultipleRegisters {
		return fmt.Errorf("功能码不一致")
	}
	return err
}

// ReadCoils读取远程设备中线圈的1到2000个连续状态，并返回线圈状态。
func (c *Client) ReadCoils(slaveID byte, address, quantity uint16) (results []byte, err error) {
	data := []byte{slaveID, FuncCodeReadCoils}
	data = append(data, uint162Bytes(address, quantity)...)
	data = crc16(data)
	c.Printf("ReadCoils: 发送[% x]", data)
	res, err := c.Send(data)
	c.Printf("ReadCoils: 接收[% x]", res)
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

// ReadDiscreteInputs 读取从1到2000连续状态的远程设备中的离散输入，并返回输入状态.
func (c *Client) ReadDiscreteInputs(slaveID byte, address, quantity uint16) (results []byte, err error) {
	data := []byte{slaveID, FuncCodeReadDiscreteInputs}
	data = append(data, uint162Bytes(address, quantity)...)
	data = crc16(data)
	c.Printf("ReadDiscreteInputs: 发送[% x]", data)
	res, err := c.Send(data)
	c.Printf("ReadDiscreteInputs: 接收[% x]", res)

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
	if res[1] != FuncCodeReadDiscreteInputs {
		return nil, fmt.Errorf("功能码不一致")
	}
	return res[3 : len(res)-2], err
}

// 日志打印
func (c *Client) Printf(format string, v ...interface{}) {
	if c.showLog {
		log.Printf(format, v...)
	}
}
