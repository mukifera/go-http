package main

import (
	"fmt"
	"net"
	"os"
	"io"
	"bytes"
	"strings"
	"regexp"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	
	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	defer conn.Close()

	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 256)

	for {
		n, err := conn.Read(tmp)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Read error:", err)
			os.Exit(1)
		}
		buf = append(buf, tmp[:n]...)
		crlf_index := bytes.Index(buf, []byte("\r\n"))
		if crlf_index != -1 {
			buf = buf[:crlf_index]
			break
		}
	}

	request_line := strings.Split(string(buf), " ")
	request_target := request_line[1]

	if matched, _ := regexp.MatchString(`^/$`, request_target); matched {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if matched, _ := regexp.MatchString(`^/echo(/\w*)*$`, request_target); matched {

		r := regexp.MustCompile(`^/echo(?:/(\w*))*$`)

		message := r.FindStringSubmatch(request_target)[1]

		fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(message), message)
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
	
}
