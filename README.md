# UDP to HTTP proxy forwarder

## Usage

```shell
udp2http [HTTP endpoint]
```

## Compile to MIPSle-32

```shell
GOOS=linux GOARCH=mipsle go build -compiler gc -o target/udp2http.$(date +'%Y.%m.%d').linux.mipsle main.go
```
