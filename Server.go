package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type userDatabase struct {
	userbase map[string]string
	mu       sync.Mutex
}

func (udb *userDatabase) set(name, pass string) {
	udb.mu.Lock()
	udb.userbase[name] = pass
	udb.mu.Unlock()
}

func (udb *userDatabase) getPass(name string) string {
	udb.mu.Lock()
	defer udb.mu.Unlock()
	return udb.userbase[name]
}

func (udb *userDatabase) getExists(name string) bool {
	udb.mu.Lock()
	defer udb.mu.Unlock()
	_, ok := udb.userbase[name]
	return ok
}

type connectedUsers struct {
	connections map[string]net.Conn
	mu          sync.Mutex
}

func (cu *connectedUsers) addUser(name string, c net.Conn) {
	cu.mu.Lock()
	cu.connections[name] = c
	cu.mu.Unlock()
}

func (cu *connectedUsers) deleteUser(name string) {
	cu.mu.Lock()
	delete(cu.connections, name)
	cu.mu.Unlock()
}

func (cu *connectedUsers) broadcast(msg string, s *server) {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	for user, con := range cu.connections {
		err := s.sendMessage(con, msg)
		if err != nil {
			s.broadcast <- user + " has disconnected.\n"
			s.users.deleteUser(user)
		}
	}
}

func (cu *connectedUsers) getExists(name string) bool {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	_, ok := cu.connections[name]
	return ok
}

type server struct {
	port      string
	l         net.Listener
	database  userDatabase
	broadcast chan string
	users     connectedUsers
}

func (s *server) init() {
	lis, err := net.Listen("tcp", s.port)
	if err != nil {
		fmt.Println(err)
	}
	s.l = lis
}

func (s *server) readMessage(c net.Conn) (string, error) {
	msg, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		return "", err
	}
	return msg, nil
}

func (s *server) sendMessage(c net.Conn, msg string) error {
	_ = c.SetWriteDeadline(time.Now().Add(time.Second * 15))
	_, err := c.Write([]byte(msg))
	if err != nil {
		return err
	}
	return nil
}

func (s *server) shutdown() {
	s.users.broadcast("SHUTDOWN\n", s)
	_ = s.l.Close()
}

func (s *server) handleConnection(c net.Conn) {
	input, err := s.readMessage(c)
	if err != nil {
		return
	}
	input = input[:len(input)-1]
	connectionFlag := true
	var username, password string
	for connectionFlag {
		username, err = s.readMessage(c)
		if err != nil {
			return
		}
		username = username[:len(username)-1]
		password, err = s.readMessage(c)
		if err != nil {
			return
		}
		password = password[:len(password)-1]
		switch input {
		case "1":
			if s.database.getPass(username) == password {
				err := s.sendMessage(c, "SUCCESS\n")
				if err != nil {
					return
				}
				connectionFlag = false
			} else {
				err := s.sendMessage(c, "FAILURE\n")
				if err != nil {
					return
				}
			}
		case "2":
			if s.database.getExists(username) {
				err := s.sendMessage(c, "FAILURE\n")
				if err != nil {
					return
				}
			} else {
				s.database.set(username, password)
				err := s.sendMessage(c, "SUCCESS\n")
				if err != nil {
					return
				}
				connectionFlag = false
			}
		}
	}
	s.users.addUser(username, c)
	s.broadcast <- username + " has entered the chat!\n"
	s.userThread(username, c)
}

func (s *server) userThread(username string, c net.Conn) {
	for {
		message, err := s.readMessage(c)
		if err != nil {
			s.broadcast <- username + " has disconnected.\n"
			s.users.deleteUser(username)
			return
		}
		if strings.TrimSpace(message) == "DISCONNECT" {
			_ = s.sendMessage(c, "DISCONNECT\n")
			s.broadcast <- username + " has disconnected.\n"
			s.users.deleteUser(username)
			break
		}
		brdMsg := username + ": " + message
		s.broadcast <- brdMsg
	}
	_ = c.Close()
}

func (s *server) broadcastThread() {
	for {
		msg := <-s.broadcast
		s.users.broadcast(msg, s)
		fmt.Print(msg)
	}
}

func (s *server) serverThread() {
	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		if strings.TrimSpace(text) == "SHUTDOWN" {
			s.shutdown()
			break
		}
		s.broadcast <- "Server: " + text
	}
}

func main() {
	arguments := os.Args
	var PORT string
	if len(arguments) == 1 {
		PORT = ":5000"
	} else {
		PORT = ":" + arguments[1]
	}
	s := server{PORT, nil, userDatabase{userbase: make(map[string]string)}, make(chan string, 16), connectedUsers{connections: make(map[string]net.Conn)}}
	s.init()
	go s.broadcastThread()
	go s.serverThread()
	for {
		c, err := s.l.Accept()
		if err != nil {
			return
		}
		go s.handleConnection(c)
	}
}
