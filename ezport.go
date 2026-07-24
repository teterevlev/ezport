// Package ezport provides a simple API for working with serial ports (COM ports) in Go.
// It automatically selects a port if not specified and allows sending data without blocking.
package ezport

import (
	"fmt"
	"sort"

	"go.bug.st/serial"
)

// Port is an independent serial port handle. Multiple Port values may be open at once.
type Port struct {
	port serial.Port
	name string
}

// Open opens a COM port.
// If portName is empty, selects the first available port alphabetically.
// Returns the actual port name that was opened.
func (p *Port) Open(portName string, baudRate int) (string, error) {
	if p.port != nil {
		return "", fmt.Errorf("port is already open (%s), call Close() first", p.name)
	}

	if portName == "" {
		ports, err := serial.GetPortsList()
		if err != nil {
			return "", err
		}
		if len(ports) == 0 {
			return "", fmt.Errorf("no COM ports found")
		}
		sort.Strings(ports)
		portName = ports[0]
	}

	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
		InitialStatusBits: &serial.ModemOutputBits{
			RTS: false,
			DTR: false,
		},
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return "", err
	}

	p.port = port
	p.name = portName
	return portName, nil
}

// Name returns the name of the opened port, or empty if not open.
func (p *Port) Name() string {
	return p.name
}

// Write sends a string to the open port.
func (p *Port) Write(message string) error {
	if p.port == nil {
		return fmt.Errorf("port is not open, call Open() first")
	}

	_, err := p.port.Write([]byte(message))
	return err
}

// Close closes the port.
func (p *Port) Close() error {
	if p.port == nil {
		return nil
	}
	err := p.port.Close()
	p.port = nil
	p.name = ""
	return err
}
