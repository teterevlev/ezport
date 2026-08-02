// Package ezport provides a simple API for working with serial ports (COM ports) in Go.
// It automatically selects a free port if not specified and allows sending data without blocking.
package ezport

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.bug.st/serial"
)

const defaultReadTimeout = 500 * time.Millisecond
const closeReadTimeout = 10 * time.Millisecond
const defaultStartReadBuf = 64
const readChunkSize = 1024

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

// Stats holds counters for the port reader.
type Stats struct {
	Dropped uint64
}

// Port is an independent serial port handle. Multiple Port values may be open at once.
type Port struct {
	mu     sync.Mutex
	port   serial.Port
	name   string
	cfg    Config
	paused bool
	wg     sync.WaitGroup // in-flight serial Read calls

	reading  bool
	stopCh   chan struct{}
	readerWG sync.WaitGroup
	dropped  atomic.Uint64
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
	p.dropped.Store(0)
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
// Cannot be used while StartRead is active.
func (p *Port) Read(buf []byte) (int, error) {
	p.mu.Lock()
	if p.reading {
		p.mu.Unlock()
		return 0, fmt.Errorf("StartRead is active, use the channel")
	}
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

// StartRead starts a background reader that sends raw chunks on a channel.
// bufSize is the channel capacity (default 64). On overflow chunks are dropped
// and Stats().Dropped is incremented; the reader never blocks on send.
// Close stops the reader and closes the channel. Not started by Open.
func (p *Port) StartRead(bufSize int) (<-chan []byte, error) {
	if bufSize <= 0 {
		bufSize = defaultStartReadBuf
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.port == nil {
		return nil, fmt.Errorf("port is not open, call Open() first")
	}
	if p.reading {
		return nil, fmt.Errorf("StartRead already running")
	}

	ch := make(chan []byte, bufSize)
	stopCh := make(chan struct{})
	p.stopCh = stopCh
	p.reading = true
	p.dropped.Store(0)

	p.readerWG.Add(1)
	go p.readLoop(ch, stopCh)

	return ch, nil
}

func (p *Port) readLoop(ch chan []byte, stopCh <-chan struct{}) {
	defer p.readerWG.Done()
	defer close(ch)
	defer func() {
		p.mu.Lock()
		p.reading = false
		if p.stopCh == stopCh {
			p.stopCh = nil
		}
		p.mu.Unlock()
	}()

	buf := make([]byte, readChunkSize)
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		p.mu.Lock()
		if p.paused || p.port == nil {
			p.mu.Unlock()
			return
		}
		port := p.port
		p.mu.Unlock()

		p.wg.Add(1)
		n, err := port.Read(buf)
		p.wg.Done()

		if err != nil {
			select {
			case <-stopCh:
				return
			default:
				continue
			}
		}
		if n <= 0 {
			continue
		}

		chunk := make([]byte, n)
		copy(chunk, buf[:n])

		select {
		case <-stopCh:
			return
		case ch <- chunk:
		default:
			p.dropped.Add(1)
		}
	}
}

// Stats returns reader counters (e.g. dropped chunks on channel overflow).
func (p *Port) Stats() Stats {
	return Stats{Dropped: p.dropped.Load()}
}

// Dropped returns the number of chunks dropped due to a full StartRead channel.
func (p *Port) Dropped() uint64 {
	return p.dropped.Load()
}

// Close closes the port and waits for an in-flight Read / StartRead to finish.
// Close on an already closed port is a no-op.
func (p *Port) Close() error {
	p.mu.Lock()
	p.paused = true
	stopCh := p.stopCh
	p.stopCh = nil
	port := p.port
	name := p.name
	p.port = nil
	p.name = ""
	if port != nil {
		_ = port.SetReadTimeout(closeReadTimeout)
	}
	p.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
	p.wg.Wait()
	p.readerWG.Wait()

	if port == nil {
		return nil
	}
	if err := port.Close(); err != nil {
		return fmt.Errorf("close %s: %w", name, err)
	}
	return nil
}
