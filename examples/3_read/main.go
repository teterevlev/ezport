package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/teterevlev/ezport"
)

func main() {
	portName := flag.String("port", "", "COM port name (empty = first free)")
	baudRate := flag.Int("baud", 115200, "Baud rate")
	bufSize := flag.Int("buf", 64, "StartRead channel capacity")
	flag.Parse()

	var p ezport.Port
	actual, err := p.Open(*portName, *baudRate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Open: %v\n", err)
		os.Exit(1)
	}
	defer p.Close()

	ch, err := p.StartRead(*bufSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "StartRead: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("reading %s (ctrl+c to stop)\n", actual)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-interrupt:
			fmt.Printf("dropped=%d\n", p.Dropped())
			return
		case <-ticker.C:
			if d := p.Dropped(); d > 0 {
				fmt.Printf("stats: dropped=%d\n", d)
			}
		case chunk, ok := <-ch:
			if !ok {
				fmt.Printf("channel closed, dropped=%d\n", p.Dropped())
				return
			}
			fmt.Printf("%s\n", hex.EncodeToString(chunk))
		}
	}
}
