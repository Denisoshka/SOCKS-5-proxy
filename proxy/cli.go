package proxy

import (
	"flag"
	"fmt"
	"os"
)

func ParseCLI() (port int) {
	flag.IntVar(&port, "port", 0, "proxy port")
	flag.Parse()

	seen := make(map[string]bool)
	flag.Visit(
		func(flag *flag.Flag) {
			seen[flag.Name] = true
		},
	)
	if !seen["port"] {
		fmt.Println("Missing required flags: -port")
		flag.PrintDefaults()
		os.Exit(1)
	}
	validatePort(port)
	return
}

func validatePort(port int) {
	if port < 0 || port > 65535 {
		fmt.Println(port, "is not valid port")
		flag.PrintDefaults()
		os.Exit(1)
	}
}
