# UDP to HTTP proxy forwarder

## Usage

```shell
udp2http [HTTP endpoint]
```

## Compile to MIPSle-32

```shell
GOOS=linux GOARCH=mipsle go build -compiler gc -o udp2http_mipsle main.go
```
