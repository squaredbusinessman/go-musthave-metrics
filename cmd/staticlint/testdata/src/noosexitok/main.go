package main

import "os"

func exit(code int) {
	os.Exit(code)
}

func main() {
	exit(1)
}
