package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var statusMessages = map[int]string{200: "OK", 201: "No Content", 404: "Not Found", 400: "Bad Request"}

type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	// Body
}

type Response struct {
	Status  int
	Headers map[string]string
	Body    string
}

func NewResponse(status int, content string, headers map[string]string) Response {
	resp := Response{Status: status, Headers: headers}
	if content != "" {
		resp.Headers["Content-Length"] = strconv.Itoa(len(content))
	}
	resp.Body = content
	return resp
}

func (r Response) joinHeaders() string {
	str := ""
	for k, v := range r.Headers {
		str += fmt.Sprintf("%v: %v\r\n", k, v)
	}
	return str
}

func (r Response) Bytes() []byte {
	return []byte(
		fmt.Sprintf("HTTP/1.1 %v %v\r\n%v\r\n%v", r.Status, statusMessages[r.Status],
			r.joinHeaders(), r.Body,
		))
}

type HTTPError struct {
	Message       string
	StatusMessage string
	Status        int
}

func (err HTTPError) Error() string {
	return fmt.Sprintf("HTTP/1.1 %v %v\r\n\r\n", err.Status, err.StatusMessage)
}


func ParseRequest(conn net.Conn) (*Request, error) {
	req := &Request{Headers: map[string]string{}}
	scanner := bufio.NewScanner(conn)
	readStart := true
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		// read start-line
		if readStart {
			requestLine := strings.Split(line, " ")
			if len(requestLine) != 3 {
				return nil, HTTPError{Status: http.StatusBadRequest, StatusMessage: "Bad Request", Message: fmt.Sprintf("Invalid start line %s", line)}
			}
			req.Method = requestLine[0]
			req.Path = requestLine[1]
			req.Version = requestLine[2]
			readStart = false
		} else {
			// read headers
			header := strings.Split(line, ": ")
			if len(header) != 2 {
				return nil, HTTPError{Status: http.StatusBadRequest, StatusMessage: "Bad Request", Message: fmt.Sprintf("Invalid Header %s", header[0])}
			}
			req.Headers[header[0]] = header[1]
		}
	}
	return req, nil
}

func handleConnection(conn net.Conn, directory string) {
	defer conn.Close()
	req, err := ParseRequest(conn)
	if err != nil {
		conn.Write([]byte(err.Error()))
		os.Exit(1)
	}
	if req.Path == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if idx := strings.Index(req.Path, "echo/"); idx != -1 {
		conn.Write(NewResponse(200, req.Path[idx+5:], map[string]string{"Content-Type": "text/plain"}).Bytes())
	} else if strings.Contains(req.Path, "/user-agent") {
		conn.Write(NewResponse(200, req.Headers["User-Agent"], map[string]string{"Content-Type": "text/plain"}).Bytes())
	} else if idx := strings.Index(req.Path, "/files"); idx != -1 {
		dirFs := os.DirFS(directory)
		fileName := req.Path[idx+7:]
		file, err := dirFs.Open(fileName)
		if err != nil {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			return
		}
		defer file.Close()
		fileData, err := io.ReadAll(file)
		if err != nil {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			return
		}
		respHeaders := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %v\r\n\r\n", len(fileData)))
		resp := append(append([]byte{},respHeaders...),fileData...)
		conn.Write(resp)
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}

func main() {
	var directory string
	if len(os.Args) == 3 {
		directory = os.Args[2]
	}
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	fmt.Println("Server Started")
	for {
		connection, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(connection, directory)
	}
}
