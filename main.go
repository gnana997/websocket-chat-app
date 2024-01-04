package main

import (
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/websocket"
)

// Struct of the channel message with from and message from the connection
type ChannelMsg struct {
	from    string
	payload []byte
}

// Struct of the server with socket connections and channel connections with channel message channel
type Server struct {
	conns        map[*websocket.Conn]bool
	channelConns map[*websocket.Conn]bool
	channelMsgCh chan ChannelMsg
}

// Creating new web socket server to establish the connections and recieve data from them
func NewServer() *Server {
	return &Server{
		conns:        make(map[*websocket.Conn]bool),
		channelConns: make(map[*websocket.Conn]bool),
		channelMsgCh: make(chan ChannelMsg, 10),
	}
}

// handle all the channel messages from connections
func (s *Server) handleWSChannel(ws *websocket.Conn) {
	fmt.Println("Incomming Connection to a channel: ", ws.RemoteAddr())

	// storing incomming connection to the channel
	s.channelConns[ws] = true

	// go routine to handle all the messages sent to the channel
	go func() {
		for msg := range s.channelMsgCh {
			fmt.Printf("Revieved message from %s : %s ", msg.from, string(msg.payload))
			// broadcast to all the channel connections with the new incomming message
			s.broadcast(msg.payload, s.channelConns)
		}
	}()

	// call the readChannelLoop function to read all the channel connections data
	s.readChannelLoop(ws)
}

// function to handle the websocket connections
func (s *Server) handleWS(ws *websocket.Conn) {
	fmt.Println("Incoming new connection from: ", ws.RemoteAddr())

	// to store all the socket connections established
	s.conns[ws] = true

	// call readLoop function to continuosly read all the data sent from the connections
	s.readLoop(ws)
}

// function to read and process the data recieved from the connections
func (s *Server) readLoop(ws *websocket.Conn) {

	// buffer to store the data from connection
	buf := make([]byte, 1024)

	for {
		n, err := ws.Read(buf)

		if err != nil {
			// to see if the connection has been ended from the remote side
			if err == io.EOF {
				fmt.Println("Connection ended from the ", ws.RemoteAddr())
				break
			}
			fmt.Println("Error occured while reading: ", err)
			continue
		}

		// to get the data sent to the server
		msg := buf[:n]

		// to broadcast the message to all the active connections
		s.broadcast(msg, s.conns)
	}
}

// function to read and process the data recieved from the channel connections
func (s *Server) readChannelLoop(ws *websocket.Conn) {

	// buffer to store the data from connection
	buf := make([]byte, 1024)

	for {
		n, err := ws.Read(buf)

		if err != nil {
			// to see if the connection has been ended from the remote side
			if err == io.EOF {
				fmt.Println("Connection ended from the ", ws.RemoteAddr())
				// to set the connection inactive
				s.channelConns[ws] = false
				break
			}
			fmt.Println("Error occured while reading: ", err)
			continue
		}

		// push the message to the channel
		s.channelMsgCh <- ChannelMsg{
			from:    ws.RemoteAddr().String(),
			payload: buf[:n],
		}
	}
}

// function to broadcast the message to all the active connections
func (s *Server) broadcast(b []byte, conns map[*websocket.Conn]bool) {
	for ws, active := range conns {
		if active {
			go func(ws *websocket.Conn) {
				if _, err := ws.Write(b); err != nil {
					fmt.Println("write error: ", err)
				}
			}(ws)
		}
	}
}

func main() {
	s := NewServer()

	http.Handle("/ws", websocket.Handler(s.handleWS))
	http.Handle("/channel", websocket.Handler(s.handleWSChannel))
	http.ListenAndServe(":3000", nil)
}
