package main

import (
	"fmt"
	"net"
	"os"
	"io"
	"bytes"
	"strings"
	"regexp"
	"flag"
	"errors"
	"log"
	"strconv"
)

func handleConnection(conn net.Conn) {

	flagSet := flag.NewFlagSet("f1", flag.ContinueOnError)
	directory_ptr := flagSet.String("directory", "", "The directory where files are stored")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing directory flag\n");
		os.Exit(1)
	}

	directory := *directory_ptr

	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 256)
	total_bytes := 0

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
		total_bytes += n
		dcrlf_index := bytes.Index(buf, []byte("\r\n\r\n"))
		if dcrlf_index != -1 {
			// buf = buf[:dcrlf_index]
			break
		}
	}

	request_line := strings.Split(string(buf), "\r\n")[0]
	headers := strings.Split(string(buf), "\r\n")[1:]

	method := strings.Split(request_line, " ")[0]
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
	} else if matched, _ := regexp.MatchString(`^/files/\w*$`, request_target); matched && method == "GET" {
		r := regexp.MustCompile(`^/files/(\w*)$`)

		file_name := r.FindStringSubmatch(request_target)[1]

		file_path := directory + "/" + file_name // need to sanitize

		contents, err := os.ReadFile(file_path)
		if errors.Is(err, os.ErrNotExist) {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		} else if err != nil {
			log.Fatal(err)
		} else {
			fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(contents), contents)
		}
	} else if matched, _ := regexp.MatchString(`^/files/\w*$`, request_target); matched && method == "POST" {
		r := regexp.MustCompile(`^/files/(\w*)$`)

		file_name := r.FindStringSubmatch(request_target)[1]

		file_path := directory + "/" + file_name // need to sanitize

		content_length := 0
		for _, header := range headers {
			split := strings.Split(header, ": ")
			name := split[0]
			value := split[1]

			if name == "Content-Length" {
				value, err := strconv.Atoi(value)
				if err != nil {
					log.Fatal(err)
				}
				content_length = value
				break
			}
		}

		dcrlf_index := bytes.Index(buf, []byte("\r\n\r\n"))

		for ; total_bytes - dcrlf_index < content_length;{
			n, err := conn.Read(tmp)
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println("Read error:", err)
				os.Exit(1)
			}
			buf = append(buf, tmp[:n]...)
			total_bytes += n
		}

		if err := os.WriteFile(file_path, buf[total_bytes-content_length :], 0644); err != nil {
			log.Fatal(err)
		}

		conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))

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
