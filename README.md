A simple HTTP server written in Go.

## How to run

Use the `server-go.sh` script to build and run the server on port 4221.

Use the `curl` command to send requests.

## Functionality
### Endpoints
- `GET /echo/{string}` echos back a string in a response body
- `GET /user-agent` reports back the `user-agent` header
- `GET /files/{filename}` passes the contents of a file in the response body. The directory of files is given to the server using the `--directory` flag
- `POST /files/{filename}` accepts text from the client and creates a new file with that text
### Supported encodings
- `gzip`
