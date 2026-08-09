package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sergio/compasso/agent/sessionlogout"
)

const operationTimeout = 10 * time.Second

func main() {
	if len(os.Args) != 1 {
		fatal(fmt.Errorf("unexpected positional arguments"))
	}

	connection, err := sessionlogout.Connect()
	if err != nil {
		fatal(err)
	}
	defer connection.Close()
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	provider, err := sessionlogout.Request(ctx, connection)
	if err != nil {
		fatal(err)
	}
	fmt.Println(provider)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "compasso-session-logout: %v\n", err)
	os.Exit(1)
}
