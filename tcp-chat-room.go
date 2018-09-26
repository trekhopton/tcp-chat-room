package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
)

// user object represents a user of the chatroom and is
// identified by a username, it is connected to a channel
type user struct {
	Username     string
	RecievedMsgs chan message
}

// message holds text in a string and comes from a user with a username
type message struct {
	Username string
	Message  string
}

// chatServer object represents the chat room containing users,
// channels for joining and leaving, and messages being passed in
type chatServer struct {
	Users    map[string]user
	Join     chan user
	Leave    chan user
	SentMsgs chan message
}

// The chatServer while running redirects sent messages,
// and handles users joining and leaving
func (cs *chatServer) Run() {
	for {
		select {
		case usr := <-cs.Join:
			cs.Users[usr.Username] = usr
			go func() {
				cs.SentMsgs <- message{
					Username: "System",
					Message:  fmt.Sprintf("*%s joined*", usr.Username),
				}
			}()
		case usr := <-cs.Leave:
			delete(cs.Users, usr.Username)
			go func() {
				cs.SentMsgs <- message{
					Username: "System",
					Message:  fmt.Sprintf("*%s left*", usr.Username),
				}
			}()
		case msg := <-cs.SentMsgs:
			for _, u := range cs.Users {
				select {
				// give user message if there's space in bufffer,
				// otherwise move on (user doesn't get message)
				case u.RecievedMsgs <- msg:
				default:
				}
			}
		}
	}
}

// connections are passed to this function for handling
func handleConnection(conn net.Conn, cs *chatServer) {
	defer conn.Close()
	// log in user from connection
	io.WriteString(conn, "Please enter your username: ")
	sc := bufio.NewScanner(conn)
	sc.Scan()
	u := user{
		Username:     sc.Text(),
		RecievedMsgs: make(chan message, 10),
	}
	cs.Join <- u
	defer func() {
		cs.Leave <- u
	}()
	// Handle messages from connection
	go func() {
		for sc.Scan() {
			cs.SentMsgs <- message{
				Username: u.Username,
				Message:  sc.Text(),
			}
		}
		cs.SentMsgs <- message{
			Username: "Disconnecting",
			Message:  u.Username,
		}
	}()
	// Deliver messages to connection
	for msg := range u.RecievedMsgs {
		if msg.Username == "Disconnecting" {
			if msg.Message == u.Username {
				break
			}
		} else {
			_, err := io.WriteString(conn, fmt.Sprintf("%s: %s\n", msg.Username, msg.Message))
			if err != nil {
				break
			}
		}
	}
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer ln.Close()

	cs := &chatServer{
		make(map[string]user),
		make(chan user),
		make(chan user),
		make(chan message),
	}
	go cs.Run()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalln(err.Error())
		}
		go handleConnection(conn, cs)
	}
}
