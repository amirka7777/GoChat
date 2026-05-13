package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)


var clients = make( map[net.Conn]string )
var mutexForClients sync.RWMutex
var logFile *os.File

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
		LogWarning("файл config.conf не найден!")
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
					LogInfo("Порт прочитан из конфига: " + value)
				case "LOG_LEVEL":
					config_default.Log_level = value
					LogInfo("Уровень логирования прочитан из конфига: " + value)
				}
				break
			}
		}
	}

	return config_default
}


func main() {

	var err error
	logFile, err = os.OpenFile("../../GoChat.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Ошибка при открытии файла GOChat.log:", err)
		os.Exit(1)
	}
	defer logFile.Close()

	config := loadConfig()

	LogInfo("Сервер запущен на порту " + config.Port)

	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		LogError("Ошибка при запуске: " + err.Error())
		os.Exit(1)
	}

	for {

		conn, err := listener.Accept() 
		if err != nil {
			LogError("Ошибка при принятии пользователя: " + err.Error())
			continue
		}

		LogInfo("Клиент подключился: " + conn.RemoteAddr().String()) 
		go handleConnection(conn)

	}

}


func handleConnection(conn net.Conn) {

	defer conn.Close()

	scanner := bufio.NewScanner(conn) 
	if !scanner.Scan() {
		return
	}
	name := scanner.Text() 

	mutexForClients.Lock()
	clients[conn] = name
	mutexForClients.Unlock()

	LogInfo("Клиент представился: " + name)
	conn.Write([]byte("Добро пожаловать в GoChat, " + name + "!\n"))

	sendMessageForClients("✋ " + name + " присоединился к чату!", conn)

	for scanner.Scan() {
		userMessage := scanner.Text()

		if userMessage == "" {
			LogWarning("[" + name + "] отправил пустое сообщение")
			continue
		}

		LogInfo("[" + name + "]: " + userMessage)
		sendMessageForClients("✉ " + "[" + name + "]" + ": " + userMessage, conn)

	}

	mutexForClients.Lock()
	delete(clients, conn)
	mutexForClients.Unlock()

	sendMessageForClients("❗ " + "[" + name + "]" + " покинул чат!", conn)	


	LogInfo("Клиент отключился: " + name)


}


func sendMessageForClients(message string, sender net.Conn) {

	mutexForClients.RLock()
	defer mutexForClients.RUnlock()

	for conn := range clients {
		if conn != sender {
			conn.Write([]byte(message + "\n"))
		}
	}

}

const (
	colorRed = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen = "\033[32m"
	colorReset = "\033[0m"
)


func LogInfo(message string) {

	timeNow := time.Now().Format("2006-01-02 15:04:05")

	fmt.Println(colorGreen + "[INFO] " + timeNow + " " + message + colorReset)
	line := "[INFO] " + timeNow + " " + message + "\n"

	if logFile != nil {
		logFile.WriteString(line)
	}

}

func LogWarning(message string) {

	timeNow := time.Now().Format("2006-01-02 15:04:05")

	fmt.Println(colorYellow + "[WARNING] " + timeNow + " " + message + colorReset)
	line := "[WARNING] " + timeNow + " " + message + "\n"

	if logFile != nil {
		logFile.WriteString(line)
	}

}

func LogError(message string) {

	timeNow := time.Now().Format("2006-01-02 15:04:05")

	fmt.Println(colorRed + "[ERROR] " + timeNow + " " + message + colorReset)
	line := "[ERROR] " + timeNow + " " + message + "\n"

	if logFile != nil {
		logFile.WriteString(line)
	}

}