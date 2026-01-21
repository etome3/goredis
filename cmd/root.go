package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

func Execute() {
	cmd := &cli.Command{
		Name:  "goredis",
		Usage: "Simplified Redis clone in Go",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "port",
				Value: "6379",
				Usage: "Port to listen on",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			port := c.String("port")

			store := NewStorage()

			listener, err := net.Listen("tcp", ":"+port)
			if err != nil {
				return fmt.Errorf("failed to bind port %s: %w\n", port, err)
			}

			fmt.Printf("Redis-Lite listening on port %s...\n", port)

			for {
				conn, err := listener.Accept()
				if err != nil {
					fmt.Printf("failed to accept connection: %v\n", err)
					continue
				}

				go handleConnection(conn, store)
			}
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func handleConnection(conn net.Conn, store *Storage) {
	defer conn.Close()

	fmt.Printf("New connection from %s\n", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		rawLine := scanner.Text()

		args := strings.Fields(rawLine)
		if len(args) == 0 {
			continue
		}

		command := strings.ToUpper(args[0])

		switch command {
		case "PING":
			conn.Write([]byte("+PONG\r\n"))

		case "ECHO":
			if len(args) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments\r\n"))
				continue
			}

			message := strings.Join(args[1:], " ")
			conn.Write([]byte("+" + message + "\r\n"))

		case "SET":
			if len(args) < 3 {
				conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
				continue
			}
			key := args[1]
			value := strings.Join(args[2:], " ")

			store.Set(key, value)
			conn.Write([]byte("+OK\r\n"))

		case "GET":
			if len(args) != 2 {
				conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
				continue
			}
			key := args[1]

			val, found := store.Get(key)
			if !found {
				conn.Write([]byte("+(nil)\r\n"))
			} else {
				conn.Write([]byte("+" + val + "\r\n"))
			}

		case "QUIT":
			conn.Write([]byte("+OK Bye\r\n"))
			return

		default:
			conn.Write([]byte("-ERR unknown command '" + command + "'\r\n"))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Connection error: %v\n", err)
	} else {
		fmt.Println("Connection closed by client")
	}
}
