package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type client struct {
	conn net.Conn
	name string
}

var (
	clients   = make(map[net.Conn]string)
	conn_port = "8080"
	mu        sync.Mutex
	muArchive sync.Mutex
	messages  = make(chan message)
)

type message struct {
	name string
	text string
}

func main() {
	if len(os.Args) > 1 && os.Args[1] != "8080" {
		conn_port = os.Args[1]
	}
	if len(os.Args) > 2 {
		log.Println("[USAGE]: ./TCPChat $port")
		return
	}
	ln, err := net.Listen("tcp", "localhost:"+conn_port)
	if err != nil {
		fmt.Println(err)
		return
	}
	go BellBoy()
	log, err := os.Create("chatArchive.txt")
	if err != nil {
		fmt.Println(err)
	}
	defer log.Close()
	fmt.Println("Listening on port " + conn_port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go HandleConnection(conn)
	}
}

func NewInput(name string, text string) message {
	return message{
		name: name,
		text: text,
	}
}

func LogoPrint() string {
	penguin, err := os.ReadFile("assets/penguin.txt")
	if err != nil {
		fmt.Println(err)
	}
	return string(penguin) + "\n"
}

func SymbCheck(s string) bool {
	for _, v := range s {
		if (v < '0' || v > '9') && (v < 'A' || v > 'Z') && (v < 'a' || v > 'z') {
			return false
		}
	}
	return true
}

func MsgCheck(s string) bool {
	for _, v := range s {
		if v <= 32 {
			return false
		}
	}
	return true
}

func CheckSame(name string) bool {
	for _, nm := range clients {
		if name == nm {
			return false
		}
	}
	return true
}

func HandleConnection(conn net.Conn) {
	logo := LogoPrint()
	conn.Write([]byte("Welcome to TCP-Chat!\n"))
	conn.Write([]byte(logo))
	var name string
	var err error

	for {
		conn.Write([]byte("[ENTER YOUR NAME]:"))
		reader := bufio.NewReader(conn)
		name, err = reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
		}
		if len(clients) > 9 {
			conn.Write([]byte("Chat is full, please wait for a free spot\n"))
			continue
		} else if name == "\n" {
			conn.Write([]byte("Please enter your name!\n"))
			continue
		} else if !CheckSame(strings.TrimSpace(name)) {
			conn.Write([]byte("Name has been already taken\n"))
			continue
		} else if !SymbCheck(strings.TrimSpace(name)) {
			conn.Write([]byte("Invalid characters! Please, enter your name using correct letters!\n"))
			continue
		} else if strings.TrimSpace(name) == "" {
			conn.Write([]byte("Please enter your name\n"))
			continue
		} else {
			break
		}
	}

	name = strings.TrimSpace(name)

	mu.Lock()
	clients[conn] = name
	mu.Unlock()

	muArchive.Lock()
	archive, err := os.OpenFile("chatArchive.txt", os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("error with archive file")
	}
	defer archive.Close()
	archiveRead, err := os.ReadFile(archive.Name())

	conn.Write(archiveRead)
	muArchive.Unlock()

	scanner := bufio.NewScanner(conn)

	muArchive.Lock()
	archive.Write([]byte(name + " has joined our chat...\n"))

	muArchive.Unlock()
	fmt.Fprintf(conn, "[%s][%s]:", time.Now().Format("2006-1-2 15:4:5"), name)
	for scanner.Scan() {
		fmt.Fprintf(conn, "[%s][%s]:", time.Now().Format("2006-1-2 15:4:5"), name)
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		} else if !MsgCheck(input) {
			// conn.Write([]byte("Invalid characters in the message!\n"))
			continue
		} else {
			userInput := NewInput(name, input)
			muArchive.Lock()
			archive.Write([]byte(("[" + time.Now().Format("2006-1-2 15:4:5") + "][" + name + "]:" + userInput.text + "\n")))
			muArchive.Unlock()
			messages <- userInput
		}

	}
	mu.Lock()
	delete(clients, conn)
	mu.Unlock()
	mu.Lock()
	messages <- NewInput(name, " has left our chat")
	mu.Unlock()
	muArchive.Lock()
	archive.Write([]byte(name + " has left our chat\n"))
	muArchive.Unlock()
	conn.Close()
}

func BellBoy() {
	for {
		select {
		case msg := <-messages:
			mu.Lock()
			for conn, name := range clients {
				if msg.name == name {
					continue
				}
				fmt.Fprintf(conn, "\n[%s][%s]:%s\n[%s][%s]:", time.Now().Format("2006-1-2 15:4:5"), msg.name, msg.text, time.Now().Format("2006-1-2 15:4:5"), name)
			}
			mu.Unlock()
		}
	}
}
