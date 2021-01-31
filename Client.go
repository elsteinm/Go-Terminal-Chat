package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

var reader = bufio.NewReader(os.Stdin)

func getMessage(c net.Conn) (string, error) {
	input, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		fmt.Println("Server is unreachable.")
		return "", err
	}
	return input, nil
}

func sendMessage(c net.Conn, msg string) error {
	_ = c.SetWriteDeadline(time.Now().Add(time.Second * 15))
	_, err := c.Write([]byte(msg))
	if err != nil {
		fmt.Println("Server is unreachable.")
		return err
	}
	return nil
}

func sendThread(c net.Conn) {
	for {
		msg, _ := reader.ReadString('\n')
		err := sendMessage(c, msg)
		if err != nil {
			fmt.Println("Server is unreachable.")
			return
		}
		if strings.TrimSpace(msg) == "DISCONNECT" {
			return
		}
	}
}

func main() {
	arguments := os.Args
	var CONNECT string
	if len(arguments) == 1 {
		fmt.Println("Please provide host:port:")
		CONNECT, _ = reader.ReadString('\n')
		CONNECT = strings.TrimSpace(CONNECT)
	} else {
		CONNECT = strings.TrimSpace(arguments[1])
	}
	c, err := net.Dial("tcp", CONNECT)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Hello! Would you like to log in or register a new account?\n1)Log in\n2)Register")
	input, _ := reader.ReadString('\n')
	err = sendMessage(c, input)
	if err != nil {
		return
	}
	connectionFlag := true
	for connectionFlag {
		fmt.Print("Username: ")
		name, _ := reader.ReadString('\n')
		err = sendMessage(c, name)
		if err != nil {
			return
		}
		fmt.Print("Password: ")
		pass, _ := reader.ReadString('\n')
		err = sendMessage(c, pass)
		if err != nil {
			return
		}
		response, err := getMessage(c)
		if err != nil {
			return
		}
		response = response[:len(response)-1]
		switch response {
		case "SUCCESS":
			fmt.Print("Welcome, ", name[:len(name)-1], "!\nType \"DISCONNECT\" to exit.\n")
			connectionFlag = false
		case "FAILURE":
			switch input {
			case "1":
				fmt.Print("Username or password incorrect. Please try again.\n")
			case "2":
				fmt.Print("Username already exists on this server. Please try again.\n")
			}
		}
	}

	go sendThread(c)

	for {
		message, err := getMessage(c)
		if err != nil {
			return
		}
		if strings.TrimSpace(message) == "SHUTDOWN" {
			fmt.Println("Server has shut down.")
			_ = c.Close()
			return
		}
		if strings.TrimSpace(message) == "DISCONNECT" {
			fmt.Println("Disconnecting...")
			_ = c.Close()
			return
		}
		fmt.Print(message)
	}
}
