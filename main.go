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
)

type UDPServerConfig struct {
	port                  int
	packetSize            int
	numProcessingHandlers int
	wg                    *sync.WaitGroup
	c                     chan os.Signal
	forwardTo             string
}

func (s *UDPServerConfig) listenAndReceive() error {
	c, err := net.ListenPacket("udp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	fmt.Printf("Starting UDP server at localhost:%d...\n", s.port)
	s.wg.Add(s.numProcessingHandlers)
	go func() {
		<-s.c
		fmt.Println("Done, exiting...")
		c.Close()
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
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		buf = bufArray[:nbytes]
		copy(buf, msg[:nbytes])
		go s.handleMessage(addr, &buf)
	}
}

func (s *UDPServerConfig) handleMessage(addr net.Addr, msg *[]byte) {
	fmt.Println(len(*msg))
	err := s.sendHTTPPost(msg)
	if err != nil {
		log.Println(err)
	}
}

func (s *UDPServerConfig) sendHTTPPost(msg *[]byte) error {
	client := http.Client{}
	rq, err := http.NewRequest("POST", s.forwardTo, bytes.NewReader(*msg))
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
		defer resp.Body.Close()
		_, err = resp.Body.Read(arr)
		if err != nil {
			arr = []byte(err.Error())
		}
		log.Printf("[%s] response body: %s", strHash, string(arr))
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Missing required parameter!\nUsage:\n\tudp2http [HTTP endpoint]")
	}

	var wg sync.WaitGroup
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT,
		syscall.SIGABRT, syscall.SIGHUP, syscall.SIGSTOP)

	udpServer := UDPServerConfig{
		port:                  20777,
		packetSize:            2048,
		numProcessingHandlers: runtime.NumCPU(),
		wg:                    &wg,
		c:                     c,
		forwardTo:             os.Args[1],
	}

	err := udpServer.listenAndReceive()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
