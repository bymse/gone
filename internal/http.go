// Package http provide simple processing of http requests
package http

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
)

type httpRequestState byte

const (
	RequestLine httpRequestState = iota
	Headers
	Body
)

type httpRequestError byte

const (
	UnsupportedMethod httpRequestError = iota
	UnsupportedHTTPVersion
	InvalidRequestLine
	MissingRequestLine
	InvalidHeader
	BodyNotSupported
	IncompleteRequest
)

type httpRequest struct {
	httpVersion   string
	method        string
	requestTarget string
	headers       map[string][]string
}

type HTTPError struct {
	code httpRequestError
}

type httpResponse struct {
	httpVersion string
	statusCode  int
	headers     map[string]string
	body        []byte
}

func (err *HTTPError) Error() string {
	switch err.code {
	case UnsupportedMethod:
		return "Unsupported method"
	case UnsupportedHTTPVersion:
		return "Unsupported http version"
	case InvalidRequestLine:
		return "Invalid request line"
	case MissingRequestLine:
		return "Missing request line"
	case InvalidHeader:
		return "Invalid header"
	case BodyNotSupported:
		return "Body not supported"
	default:
		return fmt.Sprintf("Unknown request error: %d", err.code)
	}
}

func (resp httpResponse) reasonPhrase() string {
	switch resp.statusCode {
	case 404:
		return "Not Found"
	case 400:
		return "Bad Request"
	case 403:
		return "Forbidden"
	case 200:
		return "Ok"
	default:
		return "Dunno"
	}
}

func ProcessFileGetRequest(reader io.Reader, writer io.Writer, baseDir string) error {
	req, err := parseRequest(reader)
	if err != nil {
		return writeResponse(httpResponse{
			httpVersion: "HTTP/1.1",
			statusCode:  400,
			headers:     map[string]string{},
			body:        []byte(err.Error()),
		}, writer)
	}

	path, err := url.PathUnescape(req.requestTarget)
	if err != nil {
		return writeResponse(httpResponse{
			httpVersion: req.httpVersion,
			statusCode:  400,
			body:        []byte{},
			headers:     map[string]string{},
		}, writer)
	}
	if len(path) == 1 {
		path = "."
	} else if path[0] == '/' {
		path = path[1:]
	}

	f, err := os.OpenInRoot(baseDir, path)
	if err != nil {
		return writeResponse(httpResponse{
			httpVersion: req.httpVersion,
			statusCode:  403,
			body:        []byte{},
			headers:     map[string]string{},
		}, writer)
	}

	s, err := os.Stat(f.Name())
	if err != nil {
		return writeResponse(httpResponse{
			httpVersion: req.httpVersion,
			statusCode:  404,
			body:        []byte{},
			headers:     map[string]string{},
		}, writer)
	}

	if s.IsDir() {
		return writeResponse(httpResponse{
			httpVersion: req.httpVersion,
			statusCode:  400,
			body:        []byte{},
			headers:     map[string]string{},
		}, writer)
	}

	fileBuf := make([]byte, s.Size())
	n := 0
	for {
		n, err = f.Read(fileBuf[n:])
		if err == io.EOF {
			break
		} else if err != nil {
			return writeResponse(httpResponse{
				httpVersion: req.httpVersion,
				statusCode:  500,
				body:        []byte{},
				headers:     map[string]string{},
			}, writer)
		}
	}

	return writeResponse(httpResponse{
		httpVersion: req.httpVersion,
		statusCode:  200,
		body:        fileBuf,
		headers:     map[string]string{},
	}, writer)
}

func writeResponse(resp httpResponse, writer io.Writer) error {
	buf := make([]byte, 0)

	buf = fmt.Appendf(buf, "%s %d %s\r\n", resp.httpVersion, resp.statusCode, resp.reasonPhrase())

	for key, val := range resp.headers {
		buf = fmt.Appendf(buf, "%s: %s\r\n", key, val)
	}
	buf = fmt.Append(buf, "Server: gone\r\n")

	if len(resp.body) > 0 {
		buf = fmt.Appendf(buf, "Conent-Length: %d\r\n\r\n", len(resp.body))
		buf = append(buf, resp.body...)
	} else {
		buf = fmt.Append(buf, "\r\n")
	}

	_, err := writer.Write(buf)

	return err
}

func parseRequest(reader io.Reader) (req httpRequest, err error) {
	buff := make([]byte, 4096)
	n, err := reader.Read(buff)
	req = httpRequest{}
	req.headers = make(map[string][]string)

	sectionStart := 0
	state := RequestLine
	sectionValueSet := false

	if n > 0 {
		for i := 0; i < n; i++ {
			curr := buff[i]
			var next int16 = -1
			if i < n-1 {
				next = int16(buff[i+1])
			}
			var sectionValue string
			needsSectionValue := state == RequestLine || state == Headers
			if needsSectionValue && curr == '\r' && next == '\n' {
				sectionValue = string(buff[sectionStart:i])
				sectionValueSet = true
				i++
				next = -1
				sectionStart = i + 1
			}

			switch state {
			case RequestLine:
				if !sectionValueSet {
					continue
				}

				httpErr := parseRequestLine(&req, sectionValue)
				if httpErr != nil {
					return req, httpErr
				}

				state = Headers
			case Headers:
				if !sectionValueSet {
					continue
				}

				if sectionValue == "" {
					state = Body
					continue
				}

				httpErr := parseHeader(req.headers, sectionValue)
				if httpErr != nil {
					return req, httpErr
				}
			case Body:
				return req, &HTTPError{
					code: BodyNotSupported,
				}
			}

			sectionValueSet = false
		}
	}

	if err != nil && err != io.EOF {
		return req, err
	}

	if state == Body {
		return req, nil
	}

	return req, &HTTPError{
		code: IncompleteRequest,
	}
}

func parseRequestLine(request *httpRequest, line string) *HTTPError {
	if line == "" {
		return &HTTPError{
			code: MissingRequestLine,
		}
	}

	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return &HTTPError{
			code: InvalidRequestLine,
		}
	}

	method := parts[0]
	requestTarget := parts[1]
	httpVersion := parts[2]

	if httpVersion != "HTTP/1.1" {
		return &HTTPError{
			code: UnsupportedHTTPVersion,
		}
	}

	if method != "GET" {
		return &HTTPError{
			code: UnsupportedMethod,
		}
	}

	request.httpVersion = httpVersion
	request.method = method
	request.requestTarget = requestTarget

	return nil
}

func parseHeader(headers map[string][]string, line string) *HTTPError {
	key, value, found := strings.Cut(line, ":")
	if !found {
		return &HTTPError{
			code: InvalidHeader,
		}
	}

	if key == "" || value == "" {
		return &HTTPError{
			code: InvalidHeader,
		}
	}

	headers[key] = append(headers[key], value)
	return nil
}
