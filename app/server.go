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

func handleConnection(conn net.Conn) {
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
		dcrlf_index := bytes.Index(buf, []byte("\r\n\r\n"))
		if dcrlf_index != -1 {
			buf = buf[:dcrlf_index]
			break
		}
	}

	request_line := strings.Split(string(buf), "\r\n")[0]
	headers := strings.Split(string(buf), "\r\n")[1:]
	request_target := strings.Split(request_line, " ")[1]

	if matched, _ := regexp.MatchString(`^/$`, request_target); matched {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if matched, _ := regexp.MatchString(`^/echo(/\w*)*$`, request_target); matched {

		r := regexp.MustCompile(`^/echo(?:/(\w*))*$`)

		message := r.FindStringSubmatch(request_target)[1]

		fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(message), message)
	} else if matched, _ := regexp.MatchString(`^/user-agent$`, request_target); matched {
		user_agent := ""
		for _, header := range headers {
			split := strings.Split(header, ": ")
			name := split[0]
			value := split[1]

			if name == "User-Agent" {
				user_agent = value
				break
			}
		}

		fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(user_agent), user_agent)
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
	conn.Close()
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn)
	}
	
}
