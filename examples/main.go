package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/teterevlev/ezport"
)

func main() {
	portName := flag.String("port", "", "COM port name (e.g., COM3 or /dev/ttyUSB0)")
	baudRate := flag.Int("baud", 115200, "Baud rate")
	message := flag.String("msg", "Hello, COM port!", "String to send to port")
	flag.Parse()

	var p ezport.Port
	actual, err := p.Open(*portName, *baudRate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening port: %v\n", err)
		os.Exit(1)
	}
	defer p.Close()

	if err := p.Write(*message); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to port: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Opened %s, sent: %s\n", actual, *message)
}
