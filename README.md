# UDP to HTTP proxy forwarder

## Usage

```shell
udp2http -h

NAME:
   udp2http start

USAGE:
   udp2http start [command options] [arguments...]

DESCRIPTION:
   starts UDP server that forwards requests to an HTTP endpoint defined through '-t/--target' flag

OPTIONS:
   --fs value, --frame-size value, -s value               UDP frame size to read from socket (default: 2048)
   --rqt value, --request-timeout value, --timeout value  HTTP request timeout, in seconds (default: 60)
   -p value, --port value                                 UDP socket port to start a server on (default: 20777)
   -t value, --target value                               HTTP endpoint to where forward the request
   -w value, --workers value                              number of request handing workers (default: 4)
```

## Compile to MIPSle-32

```shell
GOOS=linux GOARCH=mipsle go build -compiler gc -o target/udp2http.$(date +'%Y.%m.%d').linux.mipsle main.go
```
