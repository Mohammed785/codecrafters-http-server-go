package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var statusMessages = map[int]string{200: "OK", 201: "No Content", 404: "Not Found", 400: "Bad Request"}

type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Body    []byte
}

type Response struct {
	Status  int
	Headers map[string]string
	Body    string
}

func NewResponse(status int, content string, headers map[string]string) []byte {
	resp := Response{Status: status, Headers: headers}
	if content != "" {
		resp.Headers["Content-Length"] = strconv.Itoa(len(content))
	}
	resp.Body = content
	headerStr := joinHeaders(headers)
	b := []byte(
		fmt.Sprintf("HTTP/1.1 %v %v\r\n%v\r\n%v", status, statusMessages[status],
			headerStr, content,
		))
	return b
}

func joinHeaders(headers map[string]string) string {
	str := ""
	for k, v := range headers {
		str += fmt.Sprintf("%v: %v\r\n", k, v)
	}
	return str
}

func ParseRequest(conn net.Conn) (*Request, error) {
	req := &Request{Headers: map[string]string{}}
	readBody := false
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 256)
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			return nil, fmt.Errorf("couldn't read from connection")
		}
		buf = append(buf, tmp[:n]...)
		if n < len(tmp) {
			break
		}
	}
	for i, line := range strings.Split(string(buf), "\r\n") {
		if line == "" {
			readBody = true
			continue
		}
		// read start-line
		if i == 0 {
			requestLine := strings.Split(line, " ")
			if len(requestLine) != 3 {
				return nil, fmt.Errorf("couldn't parse request")
			}
			req.Method = requestLine[0]
			req.Path = requestLine[1]
			req.Version = requestLine[2]
		} else if readBody {
			req.Body = append(req.Body, []byte(line)...)
		} else {
			// read headers
			header := strings.Split(line, ": ")
			if len(header) != 2 {
				return nil, fmt.Errorf("couldn't parse request")
			}
			req.Headers[header[0]] = header[1]
		}
	}
	return req, nil
}
func openFile(filePath string) (*os.File, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func handleConnection(conn net.Conn, directory string) {
	defer conn.Close()
	req, err := ParseRequest(conn)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		os.Exit(1)
	}

	if req.Path == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if idx := strings.Index(req.Path, "echo/"); idx != -1 {
		conn.Write(NewResponse(200, req.Path[idx+5:], map[string]string{"Content-Type": "text/plain"}))
	} else if strings.Contains(req.Path, "/user-agent") {
		conn.Write(NewResponse(200, req.Headers["User-Agent"], map[string]string{"Content-Type": "text/plain"}))
	} else if idx := strings.Index(req.Path, "/files"); idx != -1 {
		filePath:=filepath.Join(directory, req.Path[idx+7:])
		if req.Method == "GET" {
			file, err := openFile(filePath)
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
			resp := append(append([]byte{}, respHeaders...), fileData...)
			conn.Write(resp)
		} else if req.Method == "POST" {
			file, err := os.Create(filePath)
			if err != nil {
				conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
				return
			}
			defer file.Close()
			_, e := file.Write(req.Body)
			if e != nil {
				conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
				return
			}
			conn.Write([]byte("HTTP/1.1 201 No Content\r\n\r\n"))
		}
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
		log.Fatalln("Failed to bind to port 4221")
	}
	for {
		connection, err := l.Accept()
		if err != nil {
			log.Fatalln("Error accepting connection: ", err.Error())
		}
		go handleConnection(connection, directory)
	}
}
