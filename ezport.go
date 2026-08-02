// Package ezport provides a simple API for working with serial ports (COM ports) in Go.
// It automatically selects a free port if not specified and allows sending data without blocking.
package ezport

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

const defaultReadTimeout = 500 * time.Millisecond
const closeReadTimeout = 10 * time.Millisecond

// Config holds serial port settings.
type Config struct {
	// PortName is the COM port name. Empty means auto-select the first free port.
	PortName string
	// BaudRate defaults to 9600 if <= 0.
	BaudRate int
	// ReadTimeout is applied after open. Default 500ms. Used so Read/Close do not hang forever.
	ReadTimeout time.Duration
}

func (c Config) withDefaults() Config {
	if c.BaudRate <= 0 {
		c.BaudRate = 9600
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = defaultReadTimeout
	}
	return c
}

// Port is an independent serial port handle. Multiple Port values may be open at once.
type Port struct {
	mu     sync.Mutex
	port   serial.Port
	name   string
	cfg    Config
	paused bool
	wg     sync.WaitGroup
}

func serialMode(baudRate int) *serial.Mode {
	return &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
		InitialStatusBits: &serial.ModemOutputBits{
			RTS: false,
			DTR: false,
		},
	}
}

// Open opens a COM port with the given name and baud rate.
// If portName is empty, tries ports from GetPortsList (sorted) until one opens successfully.
// Equivalent to OpenConfig(Config{PortName: portName, BaudRate: baudRate}).
func (p *Port) Open(portName string, baudRate int) (string, error) {
	return p.OpenConfig(Config{PortName: portName, BaudRate: baudRate})
}

// OpenConfig opens a COM port using Config.
// If the port is already open, it is closed first. Close is not called when nothing is open.
func (p *Port) OpenConfig(cfg Config) (string, error) {
	cfg = cfg.withDefaults()

	p.mu.Lock()
	hasPort := p.port != nil
	p.mu.Unlock()
	if hasPort {
		if err := p.Close(); err != nil {
			return "", err
		}
	}

	mode := serialMode(cfg.BaudRate)

	if cfg.PortName != "" {
		return p.openNamed(cfg.PortName, mode, cfg)
	}

	ports, err := serial.GetPortsList()
	if err != nil {
		return "", err
	}
	if len(ports) == 0 {
		return "", fmt.Errorf("no COM ports found")
	}
	sort.Strings(ports)

	var errs []string
	for _, name := range ports {
		actual, err := p.openNamed(name, mode, cfg)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		return actual, nil
	}

	return "", fmt.Errorf("no free COM ports: %s", strings.Join(errs, "; "))
}

func (p *Port) openNamed(name string, mode *serial.Mode, cfg Config) (string, error) {
	port, err := serial.Open(name, mode)
	if err != nil {
		return "", err
	}
	if err := port.SetReadTimeout(cfg.ReadTimeout); err != nil {
		_ = port.Close()
		return "", fmt.Errorf("set read timeout: %w", err)
	}

	p.mu.Lock()
	p.port = port
	p.name = name
	p.cfg = cfg
	p.paused = false
	p.mu.Unlock()
	return name, nil
}

// Name returns the name of the opened port, or empty if not open.
func (p *Port) Name() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.name
}

// Write sends a string to the open port.
func (p *Port) Write(message string) error {
	p.mu.Lock()
	port := p.port
	p.mu.Unlock()
	if port == nil {
		return fmt.Errorf("port is not open, call Open() first")
	}

	_, err := port.Write([]byte(message))
	return err
}

// Read reads up to len(buf) bytes from the port (honours ReadTimeout).
func (p *Port) Read(buf []byte) (int, error) {
	p.mu.Lock()
	if p.paused || p.port == nil {
		p.mu.Unlock()
		return 0, fmt.Errorf("port is not open, call Open() first")
	}
	port := p.port
	p.mu.Unlock()

	p.wg.Add(1)
	n, err := port.Read(buf)
	p.wg.Done()
	return n, err
}

// Close closes the port and waits for an in-flight Read to finish.
// Close on an already closed port is a no-op.
func (p *Port) Close() error {
	p.mu.Lock()
	p.paused = true
	port := p.port
	name := p.name
	p.port = nil
	p.name = ""
	if port != nil {
		_ = port.SetReadTimeout(closeReadTimeout)
	}
	p.mu.Unlock()

	p.wg.Wait()

	if port == nil {
		return nil
	}
	if err := port.Close(); err != nil {
		return fmt.Errorf("close %s: %w", name, err)
	}
	return nil
}
