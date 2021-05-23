package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"time"
)

type namedCancel struct {
	cancel context.CancelFunc
	name   string
}

var (
	command   string
	startTime = time.Now()
)

func logSinceStart() string {
	duration := int(time.Since(startTime).Seconds())
	return "[" + strconv.Itoa(duration) + "]"
}

func logf(format string, args ...interface{}) {
	_, _ = fmt.Printf(logSinceStart()+"Primary: "+format, args...)
}

func logln(msg string) {
	fmt.Println(logSinceStart(), "Primary: ", msg)
}

func runCommand(cmd *exec.Cmd, namedCancel *namedCancel) {
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		logf("Command %q exit with an error: %s\n", command, err.Error())
	} else {
		logf("Command %q exit successfully\n", command)
	}

	if namedCancel != nil {
		logf("calling cancel of %s\n", namedCancel.name)
		if namedCancel.cancel != nil {
			namedCancel.cancel()
		} else {
			logln("can't cancel because the cancel is nil")
		}
	}
}

func main() {
	flag.StringVar(&command, "command", "secondary", "The secondary command to run")
	flag.Parse()

	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	if command == "" {
		logln("command can't be empty, exit")
		os.Exit(1)
	}

	signalCtx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	logln("starting the secondary command and waiting for it to exit. If a signal comes sooner - we'll try to send an SIGINT signal to the secondary app. It it won't exit within 10 second it'll get SIGTERM and will be restarted with a shorter interval")

	var firstWarningCtx context.Context
	firstWarningCtx = context.Background()

	var finalWarningCtx context.Context
	finalWarningCtx = context.Background()

	var cmdCancel context.CancelFunc
	cmd := exec.CommandContext(mainCtx, command, "--delay", "30")
	nc := &namedCancel{
		name:   "initial cancel",
		cancel: cmdCancel,
	}

	go runCommand(cmd, nc)

BIGLOOP:
	for {
		select {
		case <-signalCtx.Done():
			logln("Got a signal let's try to stop the program")
			newSignalCtx, _ := context.WithCancel(context.Background()) // reset singal context - we don't need the 2nd signal
			signalCtx = newSignalCtx

			var firstWarningCancel context.CancelFunc
			firstWarningCtx, firstWarningCancel = context.WithTimeout(context.Background(), 10*time.Second)
			nc.name = "first warning cancel"
			nc.cancel = firstWarningCancel

			if cmd != nil {
				cmd.Process.Signal(os.Interrupt)
			}

		case <-firstWarningCtx.Done():
			if errors.Is(firstWarningCtx.Err(), context.Canceled) {
				// in case if the secondary responded to the 1st SIGINT - exit
				logln("seems line the secondary exited fast enough, exiting")
				break BIGLOOP
			}

			if errors.Is(firstWarningCtx.Err(), context.DeadlineExceeded) {
				// in case if the timer triggered after the 1st SIGINT - send SIGTERM to the secondary and restart it with a 5 sec delay,

				logln("secondary didn't finish after the 1st warning. Killing it")
				if cmd.Process == nil {
					logln("cmd.Process is nil, maybe everything is fine.. leaving")
					break BIGLOOP
				}
				if err := cmd.Process.Kill(); err != nil {
					logf("can't kill the secondary: %s\n", err.Error())
				}

				if _, err := cmd.Process.Wait(); err != nil {
					logf("something went wrong while waiting for the command to exit: %s\n", err)
				}

				logln("starting a faster version of the secondary")
				// set final warning
				var finalWarningCancel context.CancelFunc
				finalWarningCtx, finalWarningCancel = context.WithTimeout(context.Background(), 10*time.Second)

				nc.name = "final warning cancel"
				nc.cancel = finalWarningCancel

				// starting the command again
				cmd = exec.CommandContext(mainCtx, command, "--delay", "5")
				go runCommand(cmd, nc)
			}

			firstWarningCtx = context.Background()
		case <-finalWarningCtx.Done():
			if errors.Is(finalWarningCtx.Err(), context.DeadlineExceeded) {
				logln("The secondary didn't exit after the final warning. Killing it")

				err := cmd.Process.Kill()
				if err != nil {
					logf("can't kill the secondary: %s", err.Error())
				}
				break BIGLOOP
			}

			logln("seems like the secondary has exited")
			break BIGLOOP

		case <-mainCtx.Done():
			logln("Seems all done, exiting")
			break BIGLOOP
		}
	}

	logln("Primary says bye-bye to you")
}
