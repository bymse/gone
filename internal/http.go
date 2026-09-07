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
)

func ProcessRequest(reader io.Reader, writer io.Writer) error {
	buff := make([]byte, 4096)
	n, err := reader.Read(buff)
	var method string
	var requestTarget string
	var httpVersion string
	headers := make(map[string]string)

	sectionStart := 0
	state := SectionRequestLine

	if n > 0 {
		for i := 0; i < n; i++ {
			curr := buff[i]
			var next int16 = -1
			if i < n-1 {
				next = int16(buff[i+1])
			}
			var sectionValue string
			if state != SectionBody && curr == '\r' && next == '\n' {
				i++
				next = -1
				sectionValue = string(buff[sectionStart:i])
			}

			switch state {
			case SectionRequestLine:
				if sectionValue == "" {
					continue
				}
				parts := strings.Split(sectionValue, " ")
				if len(parts) != 3 {
					state = InvalidRequestLine
					continue
				}

				method = parts[0]
				requestTarget = parts[1]
				httpVersion = parts[2]

				if httpVersion != "HTTP/1.1" {
					state = UnsupportedHttpVersion
					continue
				}

				if method != "GET" {
					state = UnsupportedMethod
					continue
				}

				state = SectionHeaders
			case SectionHeaders:
				if sectionValue == "" {
					state = SectionBody
					continue
				}
			}
			sectionValue = ""
		}
	}

	return nil

}
