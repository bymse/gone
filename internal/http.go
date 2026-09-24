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

func ProcessFileGetRequest(reader io.Reader, writer io.Writer, baseDir string) error {
	req, err := parseRequest(reader)
	if err != nil {
		return err
	}

	path, err := url.PathUnescape(req.requestTarget)
	if err != nil {
		return err
	}

	f, err := os.OpenInRoot(baseDir, path)
	if err != nil {
		return err
	}

	s, err := os.Stat(f.Name())
	if err != nil {
		return err 
	}

	return nil
}

func parseRequest(reader io.Reader) (req httpRequest, err error) {
	buff := make([]byte, 4096)
	n, err := reader.Read(buff)
	httpRequest := httpRequest{}
	httpRequest.headers = make(map[string][]string)

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
				i++
				next = -1
				sectionValue = string(buff[sectionStart:i])
				sectionValueSet = true
			}

			switch state {
			case RequestLine:
				if !sectionValueSet {
					continue
				}

				httpErr := parseRequestLine(&httpRequest, sectionValue)
				if httpErr != nil {
					return req, err
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

				httpErr := parseHeader(httpRequest.headers, sectionValue)
				if httpErr != nil {
					return req, err
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
		return httpRequest, err
	}

	if state == Body {
		return httpRequest, nil
	}

	return httpRequest, &HTTPError{
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
