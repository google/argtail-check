package main

import (
	"flag"
	"fmt"
)

func foo() {
	flag.Parse()
	fmt.Println("blah")
}
