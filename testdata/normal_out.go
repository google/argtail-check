// normal is a normal program.
package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatalf("Trailing args not expected: %q", flag.Args())
	}
	fmt.Println("Hello world")
}
