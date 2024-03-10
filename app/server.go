package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	sc.Split(bufio.ScanLines)
	if !sc.Scan() {
		fmt.Printf("[Error:%s] Couldn't Read data\n", conn.RemoteAddr())
		os.Exit(1)
	}
	requestLine := strings.Split(sc.Text(), " ")
	if requestLine[1] == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}

}

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	fmt.Println("Server Started")
	connection, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}
	handleConnection(connection)
}
