package modbus

import (
	"fmt"
	"testing"
	"time"

	"github.com/tarm/serial"
)

func TestClient(t *testing.T) {
	c, err := NewClient(&serial.Config{
		Name:        "COM3",
		Baud:        9600,
		StopBits:    1,
		Parity:      serial.ParityNone,
		ReadTimeout: time.Second * 5,
	})
	c.EnableLog()
	if err != nil {
		t.Error(err)
	}
	for {
		// res, err := c.ReadCoils(10, 0, 20)
		// fmt.Println(res, err)
		res, err := c.ReadHoldingRegisters(9, 0, 2)
		fmt.Println(res, err)
		time.Sleep(time.Second * 1)
	}

}
