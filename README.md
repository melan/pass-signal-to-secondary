# Pass signal to secondary

## Why?

This small code mimics an app that runs terraform processes as the secondary. When the main program gets a signal to exit - it should make sure the terraform process exited clean and removed all locks. If it did not - it will be killed and the main process will try to run `terraform unlock` command to remove the lock.

## How to build

`make`

## How to start

```
./bin/primary --command ./bin/secondary
```

## What to expect

2 processes (primary and secondary) start and wait for some time. When primary gets SIGINT (^C) it passes this signal to the secondary and wait for some time for the secondary to exit. If it sees the secondary is running still - the primary sends SIGKILL to the secondary and restarts it with a smaller timeout. After that the primary waits for a bit. If the secondary still running after the short delay - it'll get SIGKILL from the primary and both processes will exit.

## Known problems

There is a problem with detecting exit of the secondary app after the first warning, sometimes the logs look like this:

```
[13] Primary:  secondary didn't finish after the 1st warning. Killing it
[13]Primary: something went wrong while waiting for the command to exit: wait: no child processes
[13] Primary:  starting a faster version of the secondary
[13]Primary: Command "./secondary" exit with an error: signal: killed
[13]Primary: calling cancel of final warning cancel
[13] Primary:  seems like the secondary has exited
[13] Primary:  Primary says bye-bye to you
```

The primary gets the error message from the 1st instance of the secondary after it started the 2nd instance of it. This message is recognized as the message from the 2nd instance and the whole process ends.

This is how the log should look like:

```
^C[2] Secondary:  secondary received times, setting timer to 5 seconds
[2] Primary:  Got a signal let's try to stop the program
[12] Primary:  secondary didn't finish after the 1st warning. Killing it
[12] Primary: something went wrong while waiting for the command to exit: wait: no child processes
[12] Primary:  starting a faster version of the secondary
[12] Primary: Command "./bin/secondary" exit with an error: signal: killed
[12] Primary: calling cancel of first warning cancel
[0] Secondary: stating secondary. It'll wait for 5 seconds and exit. After it gets a signal - it'll exit after 15 seconds
[5] Secondary:  timeout in the secondary, exiting
[17] Primary: Command "./bin/secondary" exit successfully
[17] Primary: calling cancel of final warning cancel
[17] Primary:  seems like the secondary has exited
[17] Primary:  Primary says bye-bye to you
```