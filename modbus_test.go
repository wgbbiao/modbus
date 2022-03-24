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
		fmt.Println(time.Now())
		res, err := c.ReadCoils(byte(01), 0, 20)
		fmt.Println(time.Now())
		fmt.Println(res, err)
	}

}
