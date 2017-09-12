package main

import (
	"flag"
	"fmt"
)

func main() {
	flag.Parse()
	fmt.Println(flag.Args())
}
