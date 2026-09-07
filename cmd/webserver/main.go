package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	http "gone/internal"
)

func main() {
	config, err := parseConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", ":"+strconv.FormatUint(uint64(config.port), 10))
	if err != nil {
		fmt.Printf("Failed to start webserver on port %d. Error: %s\n", config.port, err)
		os.Exit(1)
	}

	defer listener.Close()
	fmt.Printf("Running webserver on port %d\n", config.port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection. Error: %s", err)
			continue
		}
		http.ProcessRequest(conn, conn)
		conn.Close()
	}
}

type config struct {
	port uint16
}

func parseConfig() (config, error) {
	var port uint16 = 34876
	index := 0
	for index < len(os.Args) {
		switch os.Args[index] {
		case "--port":
			if len(os.Args) <= index+1 {
				return config{}, errors.New("missing arg for --port")
			}

			parsedPort, err := strconv.ParseUint(os.Args[index+1], 10, 16)
			if err != nil {
				msg := fmt.Sprintf("Failed to parse port %s to int", os.Args[index+1])
				return config{}, errors.Join(errors.New(msg), err)
			}

			port = uint16(parsedPort)
		}
		index++
	}

	return config{port: port}, nil
}
