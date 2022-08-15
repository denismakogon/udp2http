package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
)

type UDPServerConfig struct {
	port                  int
	packetSize            int
	numProcessingHandlers int
	wg                    *sync.WaitGroup
	c                     chan os.Signal
	target                string
}

func (s *UDPServerConfig) listenAndReceive(ctx context.Context) error {
	c, err := net.ListenPacket("udp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	log.Printf("Starting UDP server at localhost:%d...\n", s.port)
	s.wg.Add(s.numProcessingHandlers)
	go func() {
		<-s.c
		log.Println("Done, exiting...")
		_ = c.Close()
		os.Exit(0)
	}()

	for i := 0; i < s.numProcessingHandlers; i++ {
		captureOfI := i
		go func() {
			log.Printf("starting worker[%d]...\n", captureOfI)
			defer s.wg.Done()
			s.receive(ctx, c)
		}()
	}
	s.wg.Wait()
	return nil
}

func (s *UDPServerConfig) receive(ctx context.Context, c net.PacketConn) {
	msgArray := make([]byte, s.packetSize)
	bufArray := make([]byte, s.packetSize)
	var buf []byte

	msg := msgArray[0:s.packetSize]
	for {
		nbytes, addr, err := c.ReadFrom(msg)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			continue
		}
		buf = bufArray[:nbytes]
		copy(buf, msg[:nbytes])
		go s.handleMessage(ctx, addr, &buf)
	}
}

func (s *UDPServerConfig) handleMessage(ctx context.Context, addr net.Addr, msg *[]byte) {
	log.Printf("handing request from '%s'\n", addr)
	err := s.sendHTTPPost(ctx, msg)
	if err != nil {
		log.Println(err)
	}
}

func (s *UDPServerConfig) sendHTTPPost(ctx context.Context, msg *[]byte) error {
	client := http.Client{
		Timeout: time.Minute,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 60 * time.Second,
		},
	}
	rq, err := http.NewRequestWithContext(ctx, "POST", s.target, bytes.NewReader(*msg))
	if err != nil {
		return err
	}
	h := sha1.New()
	h.Write(*msg)
	strHash := hex.EncodeToString(h.Sum(nil))
	rq.Header = http.Header{
		"Validation":     {strHash},
		"Content-Type":   {"application/octet-stream"},
		"Content-Length": {strconv.Itoa(len(*msg))},
	}
	resp, err := client.Do(rq)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	log.Printf("[%s] response details: status code: %d, headers: %s\n",
		strHash, resp.StatusCode, resp.Header)
	if resp.StatusCode > 202 {
		data := make([]byte, resp.ContentLength)
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			data = []byte(err.Error())
		}
		log.Printf("[%s] response body: %s\n", strHash, string(data))
	}
	return nil
}

func main() {
	var wg sync.WaitGroup
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT,
		syscall.SIGABRT, syscall.SIGHUP, syscall.SIGSTOP)

	udpServer := UDPServerConfig{
		wg: &wg,
		c:  c,
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name: "start",
				Action: func(c *cli.Context) error {
					err := udpServer.listenAndReceive(c.Context)
					if err != nil {
						log.Fatal(err)
					}
					return nil
				},
				Description: "starts UDP server that forwards requests to an " +
					"HTTP endpoint defined through '-t/--target' flag",
				UseShortOptionHandling: true,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:        "p",
						Aliases:     []string{"port"},
						Usage:       "UDP socket port to start a server on",
						Value:       20777,
						Destination: &udpServer.port,
					},
					&cli.IntFlag{
						Name:        "fs",
						Aliases:     []string{"frame-size", "s"},
						Usage:       "UDP frame size to read from socket",
						Value:       2048,
						Destination: &udpServer.packetSize,
					},
					&cli.IntFlag{
						Name:        "w",
						Aliases:     []string{"workers"},
						Usage:       "number of request handing workers",
						Value:       runtime.NumCPU(),
						Destination: &udpServer.numProcessingHandlers,
					},
					&cli.StringFlag{
						Name:        "t",
						Aliases:     []string{"target"},
						Usage:       "HTTP endpoint to where forward the request",
						Destination: &udpServer.target,
						Required:    true,
					},
				},
			},
		},
		Name:  "udp2http",
		Usage: "UDP to HTTP traffic forwarder",
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
