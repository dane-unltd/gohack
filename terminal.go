package main

import (
	"bytes"
	"github.com/kr/pty"
	"log"
	"os"
	"os/exec"
)

type terminal struct {
	command     chan []byte
	connections map[*connection]struct{}
	register    chan *connection
	unregister  chan *connection
}

var term = terminal{
	command:     make(chan []byte),
	register:    make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]struct{}),
}

func termFunc(f *os.File, toController chan []byte, quit chan struct{}) {
	var buf [1 << 10]byte
	for {
		n, err := f.Read(buf[:])
		if err != nil {
			quit <- struct{}{}
			return
		}
		msg := make([]byte, n)
		copy(msg, buf[:n])
		toController <- msg
	}
}

func (t *terminal) run(cmdStr string) {
	cmd := exec.Command(cmdStr)
	fromPty := make(chan []byte, 10)
	quit := make(chan struct{})
	f, err := pty.Start(cmd)
	if err != nil {
		log.Fatal(err)
	}

	go termFunc(f, fromPty, quit)

	var snapshot bytes.Buffer
	for {
		select {
		case msg := <-fromPty:
			snapshot.Write(msg)
			var broadcast bytes.Buffer
			broadcast.Write([]byte("term"))
			broadcast.Write(msg)
			for c := range t.connections {
				select {
				case c.send <- broadcast.Bytes():
				default:
					//close(c.send)
					delete(t.connections, c)
				}
			}
			if snapshot.Len() > 2<<18 {
				snapshot.ReadBytes('\n')
			}
		case <-quit:
			cmd = exec.Command(cmdStr)
			f, err = pty.Start(cmd)
			if err != nil {
				log.Fatal(err)
			}

			snapshot.Reset()
			go termFunc(f, fromPty, quit)
		case c := <-t.register:
			log.Println("incoming connection")
			t.connections[c] = struct{}{}
			var msg bytes.Buffer
			msg.Write([]byte("term"))
			msg.Write(snapshot.Bytes())
			c.send <- msg.Bytes()
		case c := <-t.unregister:
			delete(t.connections, c)
			//close(c.send)
		case cmd := <-t.command:
			_, err := f.Write(cmd)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
