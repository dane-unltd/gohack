// Copyright 2013 Gary Burd. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"log"
)

type chat struct {
	message     chan []byte
	connections map[*connection]struct{}
	register    chan *connection
	unregister  chan *connection
}

var chatHub = chat{
	message:     make(chan []byte),
	register:    make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]struct{}),
}

func (ch *chat) run() {

	for {
		select {
		case c := <-ch.register:
			log.Println("incoming connection")
			ch.connections[c] = struct{}{}
		case c := <-ch.unregister:
			delete(ch.connections, c)
			//close(c.send)
		case msg := <-ch.message:
			var broadcast bytes.Buffer
			broadcast.Write([]byte("chat"))
			broadcast.Write(msg)
			for c := range ch.connections {
				select {
				case c.send <- broadcast.Bytes():
				default:
					//close(c.send)
					delete(ch.connections, c)
				}
			}
		}
	}
}
