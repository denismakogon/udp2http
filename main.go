package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
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

func (s *UDPServerConfig) listenAndReceive() error {
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
		go func() {
			defer s.wg.Done()
			s.receive(c)
		}()
	}
	s.wg.Wait()
	return nil
}

func (s *UDPServerConfig) receive(c net.PacketConn) {
	msgArray := make([]byte, s.packetSize)
	bufArray := make([]byte, s.packetSize)
	var buf []byte

	msg := msgArray[0:2048]
	for {
		nbytes, addr, err := c.ReadFrom(msg)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			continue
		}
		buf = bufArray[:nbytes]
		copy(buf, msg[:nbytes])
		go s.handleMessage(addr, &buf)
	}
}

func (s *UDPServerConfig) handleMessage(addr net.Addr, msg *[]byte) {
	log.Printf("handing request from '%s'\n", addr)
	err := s.sendHTTPPost(msg)
	if err != nil {
		log.Println(err)
	}
}

func (s *UDPServerConfig) sendHTTPPost(msg *[]byte) error {
	client := http.Client{
		Timeout: time.Minute,
	}
	rq, err := http.NewRequest("POST", s.target, bytes.NewReader(*msg))
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
	log.Printf("[%s] response details: status code: %d\n", strHash, resp.StatusCode)
	if resp.StatusCode > 202 {
		arr := make([]byte, resp.ContentLength)
		defer func() {
			_ = resp.Body.Close()
		}()
		_, err = resp.Body.Read(arr)
		if err != nil {
			arr = []byte(err.Error())
		}
		log.Printf("[%s] response body: %s", strHash, string(arr))
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
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "port",
				Aliases:     []string{"p"},
				Usage:       "UDP socket port to start a server on",
				Value:       20777,
				Destination: &udpServer.port,
			},
			&cli.IntFlag{
				Name:        "packet-size",
				Aliases:     []string{"ps"},
				Usage:       "UDP frame size to read from socket",
				Value:       2048,
				Destination: &udpServer.packetSize,
			},
			&cli.IntFlag{
				Name:        "workers",
				Aliases:     []string{"w"},
				Usage:       "number of request handing workers",
				Value:       runtime.NumCPU(),
				Destination: &udpServer.numProcessingHandlers,
			},
			&cli.StringFlag{
				Name:        "target",
				Aliases:     []string{"t"},
				Usage:       "HTTP endpoint to where forward the request",
				Destination: &udpServer.target,
				Required:    true,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	err := udpServer.listenAndReceive()
	if err != nil {
		log.Fatal(err)
	}
}
