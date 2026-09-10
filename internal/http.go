// Package http provide simple processing of http requests
package http

import (
	"io"
	"strings"
)

type httpRequestState byte

const (
	SectionRequestLine httpRequestState = iota
	SectionHeaders
	SectionBody
	UnsupportedMethod
	UnsupportedHttpVersion
	InvalidRequestLine
	MissingRequestLine
	InvalidHeader
)

type httpRequest struct {
	httpVersion   string
	method        string
	requestTarget string
	headers       map[string][]string
}

func parseRequestLine(request *httpRequest, line string) httpRequestState {
	if line == "" {
		return MissingRequestLine
	}

	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return InvalidRequestLine
	}

	method := parts[0]
	requestTarget := parts[1]
	httpVersion := parts[2]

	if httpVersion != "HTTP/1.1" {
		return UnsupportedHttpVersion
	}

	if method != "GET" {
		return UnsupportedMethod
	}

	request.httpVersion = httpVersion
	request.method = method
	request.requestTarget = requestTarget

	return SectionHeaders
}

func parseHeader(headers map[string][]string, line string) httpRequestState {
	key, value, found := strings.Cut(line, ":")
	if !found {
		return InvalidHeader
	}

	if key == "" || value == "" {
		return InvalidHeader
	}

	headers[key] = append(headers[key], value)
	return SectionHeaders
}

func ProcessRequest(reader io.Reader, writer io.Writer) error {
	buff := make([]byte, 4096)
	n, err := reader.Read(buff)
	httpRequest := httpRequest{}
	httpRequest.headers = make(map[string][]string)

	sectionStart := 0
	state := SectionRequestLine
	sectionValueSet := false

	if n > 0 {
		for i := 0; i < n; i++ {
			curr := buff[i]
			var next int16 = -1
			if i < n-1 {
				next = int16(buff[i+1])
			}
			var sectionValue string
			needsSectionValue := state == SectionRequestLine || state == SectionHeaders
			if needsSectionValue && curr == '\r' && next == '\n' {
				i++
				next = -1
				sectionValue = string(buff[sectionStart:i])
				sectionValueSet = true
			}

			switch state {
			case SectionRequestLine:
				if !sectionValueSet {
					continue
				}
				state = parseRequestLine(&httpRequest, sectionValue)
			case SectionHeaders:
				if !sectionValueSet {
					continue
				}

				if sectionValue == "" {
					state = SectionBody
					continue
				}

				state = parseHeader(httpRequest.headers, sectionValue)
			}
			sectionValueSet = false
		}
	}

	return nil

}
