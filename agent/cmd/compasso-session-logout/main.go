package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sergio/compasso/agent/sessionlogout"
)

const operationTimeout = 10 * time.Second

func main() {
	probeOnly := flag.Bool("probe", false, "detect an orderly logout provider without logging out")
	flag.Parse()
	if flag.NArg() != 0 {
		fatal(fmt.Errorf("unexpected positional arguments"))
	}

	connection, err := sessionlogout.Connect()
	if err != nil {
		fatal(err)
	}
	defer connection.Close()
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	var provider string
	if *probeOnly {
		provider, err = sessionlogout.Detect(ctx, connection)
	} else {
		provider, err = sessionlogout.Request(ctx, connection)
	}
	if err != nil {
		fatal(err)
	}
	fmt.Println(provider)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "compasso-session-logout: %v\n", err)
	os.Exit(1)
}
