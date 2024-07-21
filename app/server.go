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
	"compress/gzip"
)

type Request struct {
	method string
	target string
	headers map[string]string
	body []byte

	total_bytes_read int
	buf []byte
	tmp []byte
}

type Response struct {
	status string
	headers map[string]string
	body []byte
}

func parseRequestFromConnection(conn net.Conn) Request {
	var request Request

	request.total_bytes_read = 0
	request.buf = make([]byte, 0, 4096)
	request.headers = make(map[string]string)
	tmp := make([]byte, 256)

	var request_line string

	for {
		n, err := conn.Read(tmp)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Read error:", err)
			os.Exit(1)
		}
		request.buf = append(request.buf, tmp[:n]...)
		request.total_bytes_read += n

		dcrlf_index := bytes.Index(request.buf, []byte("\r\n\r\n"))
		if dcrlf_index != -1 {
			request_line = strings.Split(string(request.buf[:dcrlf_index]), "\r\n")[0]
			header_strings := strings.Split(string(request.buf[:dcrlf_index]), "\r\n")[1:]
			for _, header_string := range header_strings {
				split := strings.Split(header_string, ": ")
				name := split[0]
				value := split[1]
	
				request.headers[name] = value
			}

			break
		}
	}

	request.method = strings.Split(request_line, " ")[0]
	request.target = strings.Split(request_line, " ")[1]

	return request
}

func parseRequestBody(request *Request, conn net.Conn) {
	content_length_string, ok := request.headers["Content-Length"]
	if !ok {
		log.Fatal("No Content-Length header was provided")
	}
	content_length, err := strconv.Atoi(content_length_string)
	if err != nil {
		log.Fatal(err)
	}

	dcrlf_index := bytes.Index(request.buf, []byte("\r\n\r\n"))

	tmp := make([]byte, 256)
	for ; request.total_bytes_read - dcrlf_index - 4 < content_length;{
		n, err := conn.Read(tmp)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Read error:", err)
			os.Exit(1)
		}
		request.buf = append(request.buf, tmp[:n]...)
		request.total_bytes_read += n
	}

	request.body = request.buf[request.total_bytes_read-content_length :]
}

func newResponse() Response {
	var response Response
	response.headers = make(map[string]string)
	return response
}

func compressGzip(message []byte) []byte {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	_, err := writer.Write(message)
	if err != nil {
		log.Fatal(err)
	}
	writer.Close()
	fmt.Println(string(buffer.Bytes()))
	return buffer.Bytes()
}

func compressResponse(response *Response, encodings_string string) {
	encodings := strings.Split(encodings_string, ", ")
	var response_encodings []string
	for _, encoding := range encodings {
		if encoding == "gzip" {
			response_encodings = append(response_encodings, "gzip")
			response.body = compressGzip(response.body)
			response.headers["Content-Length"] = strconv.Itoa(len(response.body))
		}
	}

	if len(response_encodings) == 0 {
		return
	}

	response.headers["Content-Encoding"] = strings.Join(response_encodings, ", ")
}

func handleConnection(conn net.Conn) {

	flagSet := flag.NewFlagSet("f1", flag.ContinueOnError)
	directory_ptr := flagSet.String("directory", "", "The directory where files are stored")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing directory flag\n");
		os.Exit(1)
	}

	directory := *directory_ptr

	request := parseRequestFromConnection(conn)
	response := newResponse()

	if matched, _ := regexp.MatchString(`^/$`, request.target); matched {
		response.status = "200 OK"
	} else if matched, _ := regexp.MatchString(`^/echo(/\w*)*$`, request.target); matched {

		r := regexp.MustCompile(`^/echo(?:/(\w*))*$`)

		message := r.FindStringSubmatch(request.target)[1]

		response.status = "200 OK"
		response.headers["Content-Type"] = "text/plain"
		response.headers["Content-Length"] = strconv.Itoa(len(message))
		response.body = []byte(message)

	} else if matched, _ := regexp.MatchString(`^/user-agent$`, request.target); matched {
		user_agent, ok := request.headers["User-Agent"]
		if !ok {
			fmt.Fprintf(os.Stderr, "No User-Agent header was provided\n");
			
			response.status = "400 Bad Request"
		} else {
			response.status = "200 OK"
			response.headers["Content-Type"] = "text/plain"
			response.headers["Content-Length"] = strconv.Itoa(len(user_agent))
			response.body = []byte(user_agent)
		}
	} else if matched, _ := regexp.MatchString(`^/files/\w*$`, request.target); matched && request.method == "GET" {
		r := regexp.MustCompile(`^/files/(\w*)$`)

		file_name := r.FindStringSubmatch(request.target)[1]

		file_path := directory + "/" + file_name // need to sanitize

		contents, err := os.ReadFile(file_path)
		if errors.Is(err, os.ErrNotExist) {
			response.status = "404 Not Found"
		} else if err != nil {
			log.Fatal(err)
		} else {
			response.status = "200 OK"
			response.headers["Content-Type"] = "application/octet-stream"
			response.headers["Content-Length"] = strconv.Itoa(len(contents))
			response.body = contents
		}
	} else if matched, _ := regexp.MatchString(`^/files/\w*$`, request.target); matched && request.method == "POST" {
		r := regexp.MustCompile(`^/files/(\w*)$`)

		file_name := r.FindStringSubmatch(request.target)[1]

		file_path := directory + "/" + file_name // need to sanitize

		parseRequestBody(&request, conn)

		if err := os.WriteFile(file_path, request.body, 0644); err != nil {
			log.Fatal(err)
		}
		response.status = "201 Created"

	} else {
		response.status = "404 Not Found"
	}

	if encodings, ok := request.headers["Accept-Encoding"]; ok && request.method == "GET" {
		compressResponse(&response, encodings)
	}

	fmt.Fprintf(conn, "HTTP/1.1 %s\r\n", response.status)
	for key, value := range response.headers {
		fmt.Fprintf(conn, "%s: %s\r\n", key, value)
	}
	fmt.Fprintf(conn, "\r\n%s", response.body)

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
