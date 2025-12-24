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
    "flag"
    "github.com/teterevlev/ezport"
)

func main() {
    portName := flag.String("port", "", "COM port name")
    baudRate := flag.Int("baud", 9600, "Baud rate")
    message := flag.String("msg", "Hello, COM port!", "Message")
    flag.Parse()

    ezport.Open(portName, baudRate)
    ezport.Write(*message)
    defer ezport.Close()
}
```

## Features

- ✅ **Automatic port selection** - if port is not specified, selects the first one alphabetically
- ✅ **Simple API** - just two functions: `Open()` and `Write()`
- ✅ **Error handling** - errors are printed to stderr without crashing the program
- ✅ **Cross-platform** - Windows, Linux, macOS

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
