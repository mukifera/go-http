package main

import (
	"fmt"
	"net"
	"os"
	"io"
	"bytes"
	"strings"
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

	// defer conn.Close()

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

	if len(request_line) == 3 && request_line[1] != "/" {
		fmt.Println(request_line[1])
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	} else {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	}

	
}
