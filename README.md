# UDP to HTTP proxy forwarder

## Usage

```shell
udp2http -h
NAME:
   start - UDP to HTTP traffic forwarder

USAGE:
   start [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --fs value, --frame-size value, -s value  UDP frame size to read from socket (default: 2048)
   --help, -h                                show help (default: false)
   -p value, --port value                    UDP socket port to start a server on (default: 20777)
   -t value, --target value                  HTTP endpoint to where forward the request
   -w value, --workers value                 number of request handing workers (default: 4)
   
```

## Compile to MIPSle-32

```shell
GOOS=linux GOARCH=mipsle go build -compiler gc -o target/udp2http.$(date +'%Y.%m.%d').linux.mipsle main.go
```
