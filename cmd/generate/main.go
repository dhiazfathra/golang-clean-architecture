package main

import (
	"os"
)

func main() {
	deps := defaultDeps()
	code := serve(os.Args[1:], deps)
	if code != 0 {
		deps.Exit(code)
	}
}
