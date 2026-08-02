# ezport - Simple COM Port Module for Go

Easy-to-use module for working with serial ports (COM ports) in Go. Automatically selects a port if not specified and provides a simple API for sending data.

## Installation

```bash
go get github.com/teterevlev/ezport
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/teterevlev/ezport"
)

func main() {
    var p ezport.Port
    name, err := p.Open("", 9600) // empty name → first free port
    if err != nil {
        panic(err)
    }
    defer p.Close()

    fmt.Println("opened", name)
    _ = p.Write("Hello, COM port!")
}
```

Or with explicit config (read timeout, etc.):

```go
var p ezport.Port
name, err := p.OpenConfig(ezport.Config{
    PortName:    "",
    BaudRate:    115200,
    ReadTimeout: 500 * time.Millisecond,
})
```

Two independent ports (explicit names):

```go
var p1, p2 ezport.Port
_, err1 := p1.Open("COM3", 9600)
_, err2 := p2.Open("COM4", 115200)
```

Two ports with auto-select (next free each time):

```go
var p1, p2 ezport.Port
name1, err1 := p1.Open("", 9600)
name2, err2 := p2.Open("", 9600) // skips busy; errors if no free port left
```

## Features

- ✅ **Instance-based API** — each `Port` is an independent handle (multiple ports at once)
- ✅ **Automatic port selection** — if port name is empty, opens the first free port (busy skipped)
- ✅ **Config** — baud, read timeout; 8N1 with RTS/DTR off
- ✅ **Safe Close** — waits for an in-flight `Read`; no-op if already closed
- ✅ **Simple API** — `Open` / `OpenConfig`, `Write`, `Read`, `Close`
- ✅ **Errors are returned** — caller decides how to handle them
- ✅ **Cross-platform** — Windows, Linux, macOS

## Breaking change

Package-level `Open` / `Write` / `Close` and `Open(*string, *int)` were replaced by methods on `Port`:

```go
var p ezport.Port
actual, err := p.Open(portName, baudRate)
```

## Usage Examples

```bash
# Send a message (auto-select or -port)
go run ./examples/1_send
go run ./examples/1_send -port COM3
go run ./examples/1_send -port COM3 -baud 115200 -msg "Hello!"

# Two auto-selected free ports (needs ≥2 free COM ports)
go run ./examples/2_two_ports
```

## Dependencies

- [go.bug.st/serial](https://github.com/bugst/go-serial) - cross-platform library for working with serial ports

## License

MIT
