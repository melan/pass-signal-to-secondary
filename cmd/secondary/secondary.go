package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"
)

var (
	delay      int
	quickDelay = 15
	startTime  = time.Now()
)

func getSignalCtx(ctx context.Context) context.Context {
	signalCtx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	return signalCtx
}

func logSinceStart() string {
	duration := int(time.Since(startTime).Seconds())
	return "[" + strconv.Itoa(duration) + "]"
}

func logf(format string, args ...interface{}) {
	fmt.Printf(logSinceStart()+" Secondary: "+format, args...)
}

func logln(msg string) {
	fmt.Println(logSinceStart(), "Secondary: ", msg)
}

func main() {
	flag.IntVar(&delay, "delay", 30, "This is a delay for the secondary to wait")

	flag.Parse()

	if delay <= 0 {
		delay = 30
	}

	logf("stating secondary. It'll wait for %d seconds and exit. After it gets a signal - it'll exit after %d seconds\n", delay, quickDelay)

	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	signalCtx := getSignalCtx(context.Background())

	timer := time.After(time.Duration(delay) * time.Second)
	signalTimer := time.After(time.Duration(delay*10) * time.Second)

	for {
		select {
		case <-mainCtx.Done():
			logln("main context of the secondary is canceled, exiting")
			return
		case <-timer:
			logln("timeout in the secondary, exiting")
			return
		case <-signalTimer:
			logln("secondary is done waiting for 5 seconds after the signal, exiting")
			return
		case <-signalCtx.Done():
			logln("secondary received times, setting timer to 5 seconds")
			var c context.CancelFunc
			signalCtx, c = context.WithCancel(context.Background()) // reset signal context
			defer c()

			signalTimer = time.After(time.Duration(quickDelay) * time.Second)
			continue
		}
	}
}
