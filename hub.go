// Copyright 2013 Gary Burd. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/kr/pty"
	"log"
	"os/exec"
)

type hub struct {
	connections map[*connection]bool
	command     chan []byte
	register    chan *connection
	unregister  chan *connection
}

var h = hub{
	command:     make(chan []byte),
	register:    make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]bool),
}

func (h *hub) run() {
	cmd := exec.Command("nethack")
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
			for c := range h.connections {
				select {
				case c.send <- msg:
				default:
					close(c.send)
					delete(h.connections, c)
				}
			}
		case c := <-h.register:
			log.Println("incoming connection")
			h.connections[c] = true
		case c := <-h.unregister:
			delete(h.connections, c)
			close(c.send)
		case cmd := <-h.command:
			_, err := f.Write(cmd)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
