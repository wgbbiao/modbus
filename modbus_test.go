package main

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
	{
		// res, err := c.ReadCoils(10, 0, 20)
		// fmt.Println(res, err)
		err := c.WriteSingleCoil(10, 4, false)
		fmt.Println(err)
	}

}
