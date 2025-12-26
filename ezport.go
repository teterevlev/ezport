// Package ezport provides a simple API for working with serial ports (COM ports) in Go.
// It automatically selects a port if not specified and allows sending data without blocking.
package ezport

import (
	"fmt"
	"sort"

	"go.bug.st/serial"
)

var port serial.Port

// Open opens a COM port. If portName is empty, selects the first available port alphabetically.
func Open(portName *string, baudRate *int) error {
	// If port is not specified, select the first one alphabetically
	if *portName == "" {
		ports, err := serial.GetPortsList()
		if err != nil {
			return err
		}
		if len(ports) == 0 {
			return fmt.Errorf("no COM ports found")
		}
		sort.Strings(ports)
		*portName = ports[0]
	}

	// Port settings - disable flow control to avoid waiting
	mode := &serial.Mode{
		BaudRate: *baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
		InitialStatusBits: &serial.ModemOutputBits{
			RTS: false,
			DTR: false,
		},
	}

	// Open the port
	var err error
	port, err = serial.Open(*portName, mode)
	if err != nil {
		return err
	}

	return nil
}

// Write sends a string to the open port.
func Write(message string) error {
	if port == nil {
		return fmt.Errorf("port is not open, call Open() first")
	}

	_, err := port.Write([]byte(message))
	if err != nil {
		return err
	}

	return nil
}

// Close closes the port.
func Close() error {
	if port == nil {
		return nil
	}
	err := port.Close()
	port = nil
	return err
}

