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
    name, err := p.Open("", 9600) // empty name → first port alphabetically
    if err != nil {
        panic(err)
    }
    defer p.Close()

    fmt.Println("opened", name)
    _ = p.Write("Hello, COM port!")
}
```

Two independent ports:

```go
var p1, p2 ezport.Port
_, err1 := p1.Open("COM3", 9600)
_, err2 := p2.Open("COM4", 115200)
```

## Features

- ✅ **Instance-based API** — each `Port` is an independent handle (multiple ports at once)
- ✅ **Automatic port selection** — if port name is empty, selects the first one alphabetically
- ✅ **Simple API** — `Open`, `Write`, `Close`
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
# Run the example from project root
go run ./examples/main.go

# Specify a specific port
go run ./examples/main.go -port COM3

# With baud rate and message
go run ./examples/main.go -port COM3 -baud 115200 -msg "Hello!"
```

## Dependencies

- [go.bug.st/serial](https://github.com/bugst/go-serial) - cross-platform library for working with serial ports

## License

MIT
