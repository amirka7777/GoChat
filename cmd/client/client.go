package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

type Config struct {
	Port string
	Log_level string
}

func loadConfig() *Config {

	config_default := &Config{
		Port: "8080",
		Log_level: "info",
	}

	file, err := os.Open("../../config.conf")
	if err != nil {
		return config_default
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 || line[0] == '#' {
			continue
		}

		for i := 0; i < len(line); i++ {
			if line[i] == '=' {
				key := line[:i]
				value := line[i+1:]

				switch key {
				case "PORT":
					config_default.Port = value
				case "LOG_LEVEL":
					config_default.Log_level = value
				}
				break
			}
		}
	}

	return config_default
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Ошибка при запуске: (подсказка) go run client.go <имя>")
		return
	}

	config := loadConfig()

	name := os.Args[1]

	conn, err := net.Dial("tcp", "localhost:"+config.Port)
	if err != nil {
		fmt.Println("Ошибка при подключении:", err)
		os.Exit(1)
	}
	defer conn.Close()

	conn.Write([]byte(name + "\n"))

	go acceptanceMessage(conn)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		userText := scanner.Text()
		conn.Write([]byte(userText + "\n"))
		if userText == "/exit" {
			break
		}
	}



}

func acceptanceMessage(conn net.Conn) {

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

}