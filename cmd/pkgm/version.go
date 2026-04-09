package main

import "fmt"

var version = "dev"

func cmdVersion() {
	fmt.Println("pkgm " + version)
}
