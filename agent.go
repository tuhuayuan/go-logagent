package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// RunAgent run agent mode.
func runAgent() int {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		for {
			time.Sleep(1 * time.Second)
		}
	}()
	for {
		fmt.Println("agent waiting.")
		select {
		case sig := <-signalChan:
			fmt.Println("agent get ", sig)
			return 0
		}
	}
}
