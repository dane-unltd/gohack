package main

import (
	"bytes"
	"github.com/kr/pty"
	"log"
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

func (t *terminal) run(cmdStr string) {
	cmd := exec.Command(cmdStr)
	fromPty := make(chan []byte, 10)
	f, err := pty.Start(cmd)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		var buf [1 << 10]byte
		for {
			n, err := f.Read(buf[:])
			if err != nil {
				log.Fatal(err)
			}
			msg := make([]byte, n)
			copy(msg, buf[:n])
			fromPty <- msg
		}
	}()

	for {
		select {
		case msg := <-fromPty:
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
		case c := <-t.register:
			log.Println("incoming connection")
			t.connections[c] = struct{}{}
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
