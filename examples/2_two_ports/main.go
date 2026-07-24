package main

import (
	"fmt"
	"os"

	"github.com/teterevlev/ezport"
)

// Opens two ports with empty names: each takes the next free COM port.
// Needs at least two free ports on the machine.
func main() {
	var p1, p2 ezport.Port

	name1, err := p1.Open("", 9600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "first Open: %v\n", err)
		os.Exit(1)
	}
	defer p1.Close()

	name2, err := p2.Open("", 9600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "second Open: %v\n", err)
		os.Exit(1)
	}
	defer p2.Close()

	fmt.Printf("opened %s and %s\n", name1, name2)
}
