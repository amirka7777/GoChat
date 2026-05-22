package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)


var clients = make( map[net.Conn]string )
var mutexForClients sync.RWMutex
var clientRoom = make( map[net.Conn]*Room )
var mutexForClientRoom sync.RWMutex
var globalConfig *Config

var logFile *os.File

type Config struct {
	Port string
	InactivityTimer int
	MaxUserInRoom int
}

type Room struct {
	Name string
	Password string
	Clients map[net.Conn]string
	mu sync.RWMutex
}

var rooms = make(map[string]*Room)
var mutexForRooms sync.RWMutex

func CreateRoom(name, password string) *Room {

	room := &Room{
		Name: name,
		Password: password,
		Clients: make(map[net.Conn]string),
	}

	mutexForRooms.Lock()
	rooms[name] = room
	mutexForRooms.Unlock()

	LogInfo("Создана комната: " + name + " (пароль: " + strconv.FormatBool(RoomHavePassword(room.Password)) +  ")")
	return room


}

func RoomHavePassword(s string) bool {
	return s != ""
}

func (r *Room) AddClient(conn net.Conn, name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.Clients) >= globalConfig.MaxUserInRoom {
		return false
	}

	r.Clients[conn] = name
	return true
}

func (r *Room) RemoveClient(conn net.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Clients, conn)
}

func (r *Room) sendMessageForRoom(sender net.Conn, msg string) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	for conn := range r.Clients {
		if conn != sender {
			conn.Write([]byte(msg + "\n"))
		}
	}

}

func (r *Room) CheckCountPeople() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Clients)
}

func (r *Room) isFull() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Clients) >= globalConfig.MaxUserInRoom
}

func (r *Room) HavePassword() bool {
	return r.Password != ""
}

func (r *Room) CheckPassword(password string) bool {
	if r.Password == "" {
		return true
	}

	return r.Password == password
}

func loadConfig() *Config {

	config_default := &Config{
		Port: "8080",
		InactivityTimer: 300,
		MaxUserInRoom: 20,
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
				case "INACTIVITY_TIMER":
					time_inactivity, _ := strconv.Atoi(value)
					if time_inactivity > 0 {
						config_default.InactivityTimer = time_inactivity
						LogInfo("Время бездействия прочитано из конфига: " + value)
					}

				case "MAX_USERS_IN_ROOM":
					maxUsers, _ := strconv.Atoi(value)
					if maxUsers > 0 {
						config_default.MaxUserInRoom = maxUsers
						LogInfo("Максимальное количество людей прочитано из конфига: " + value)
					}
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

	
	globalConfig = loadConfig()

	CreateRoom("lobby", "")
	LogInfo("Создана общая комната 'lobby'")

	LogInfo("Сервер запущен на порту " + globalConfig.Port)

	listener, err := net.Listen("tcp", ":"+globalConfig.Port)
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

	var idleTimer *time.Timer
	if globalConfig.InactivityTimer > 0 {
		idleTimer = time.NewTimer(time.Duration(globalConfig.InactivityTimer) * time.Second)
		go func () {
			<- idleTimer.C
			LogWarning("Пользователь " + name + " отключен за бездействие!")
			conn.Write([]byte("Вы отключены за бездействие!\n"))
			conn.Close()
		}()
	}

	mutexForClients.Lock()
	clients[conn] = name
	mutexForClients.Unlock()

	lobby := rooms["lobby"]
	if lobby == nil {
		LogError("Комната lobby не найдена!")
		return 
	}

	if !lobby.AddClient(conn, name) {
		LogError("Не удалось добавить клиента в lobby: комната переполнена")
		conn.Write([]byte("Комната lobby переполнена! Максимум: " + strconv.Itoa(globalConfig.MaxUserInRoom) + " человек!\n"))
		return
	}

	mutexForClientRoom.Lock()
	clientRoom[conn] = lobby
	mutexForClientRoom.Unlock()

	LogInfo("Клиент представился: " + name)
	conn.Write([]byte("Добро пожаловать в GoChat, " + name + "!\n"))
	conn.Write([]byte("Ваше текущее положение - общая комната lobby!\n"))

	lobby.sendMessageForRoom(conn, "✋ " + name + " присоединился к чату!")

	for scanner.Scan() {
		userMessage := scanner.Text()

		if idleTimer != nil {
			idleTimer.Reset(time.Duration(globalConfig.InactivityTimer) * time.Second)
		}

		if userMessage == "" {
			LogWarning("[" + name + "] отправил пустое сообщение")
			continue
		}

		if len(userMessage) > 0 && userMessage[0] == '/' {
			handleCommand(conn, name, userMessage)
		} else {
			
			mutexForClientRoom.RLock()
			currentRoom := clientRoom[conn]
			mutexForClientRoom.RUnlock()

			if currentRoom != nil {
				LogInfo("[" + name + " -> " + currentRoom.Name + "]: " + userMessage)
				currentRoom.sendMessageForRoom(conn, "✉ ["+name+"]: "+ userMessage)
			}

		}

	}
	if idleTimer != nil {
		idleTimer.Stop()
	}

	mutexForClientRoom.RLock()
	currentRoom := clientRoom[conn]
	mutexForClientRoom.RUnlock()

	if currentRoom != nil {
		currentRoom.RemoveClient(conn)
		currentRoom.sendMessageForRoom(conn, "❗ " + "[" + name + "]" + " покинул чат!")
	}

	mutexForClients.Lock()
	delete(clients, conn)
	mutexForClients.Unlock()

	mutexForClientRoom.Lock()
	delete(clientRoom, conn) 
	mutexForClientRoom.Unlock()


	LogInfo("Клиент отключился: " + name)


}

func handleCommand(conn net.Conn, name, fullCommand string) {

	parts := splitCommand(fullCommand)
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	argumets := parts[1:]

	switch command {
	case "/create":
		if len(argumets) < 1 {
			conn.Write([]byte("❌ Использование: /create <название> [пароль]\n"))
			return
		}

		roomName := argumets[0]
		password := ""
		if len(argumets) >= 2 {
			password = argumets[1]
		}

		mutexForRooms.RLock()
		_, existsRoom := rooms[roomName]
		mutexForRooms.RUnlock()

		if existsRoom != false {
			conn.Write([]byte("❌ Комната " + roomName + " уже существует!\n"))
			return
		}

		CreateRoom(roomName, password)
		conn.Write([]byte("✅ Комната " + roomName + " успешно создана!\n"))
	case "/join":
		if len(argumets) < 1 {
			conn.Write([]byte("❌ Использование: /join <название> [пароль]\n"))
			return
		}

		roomName := argumets[0]
		password := ""
		if len(argumets) >= 2 {
			password = argumets[1]
		}

		mutexForRooms.RLock()
		checkRoom, existsRoom := rooms[roomName]
		mutexForRooms.RUnlock()

		if existsRoom == false {
			conn.Write([]byte("❌ Комната " + roomName + " не существует!\n"))
			return 
		}

		if checkRoom.isFull() {
			conn.Write([]byte("Комната " + roomName + " переполнена! Максимум: " + strconv.Itoa(globalConfig.MaxUserInRoom) + " человек!\n"))
			LogWarning("Неудачная попытка входа в комнату " + roomName + " от пользователя " + name)
			return 
		}

		if !checkRoom.CheckPassword(password) {
			conn.Write([]byte("❌ Неверный пароль для комнаты " + roomName + "\n"))
			LogWarning("Неудачная попытка входа в комнату " + roomName + " от пользователя " + name)
			return 
		}

		mutexForClientRoom.RLock()
		oldRoom := clientRoom[conn]
		mutexForClientRoom.RUnlock()

		if oldRoom != nil && oldRoom.Name == roomName {
			conn.Write([]byte("❌ Вы уже в данной комнате!\n"))
			return
		}

		if oldRoom != nil {
			oldRoom.RemoveClient(conn)
			oldRoom.sendMessageForRoom(conn, "❗ [" + name + "] покинул комнату " + oldRoom.Name)
		}

		checkRoom.AddClient(conn, name)

		mutexForClientRoom.Lock()
		clientRoom[conn] = checkRoom
		mutexForClientRoom.Unlock()

		conn.Write([]byte("✅ Вы успешно вошли в комнату " + roomName + "\n"))
		checkRoom.sendMessageForRoom(conn, "✋ " + name + " присоединился к комнате!")
		if oldRoom != nil {
    		LogInfo("[" + name + "]: перешел из комнаты " + oldRoom.Name + " в " + checkRoom.Name)
		} else {
    		LogInfo("[" + name + "]: вошел в комнату " + checkRoom.Name)
		}
	
	case "/leave":

		mutexForClientRoom.RLock()
		currentRoom := clientRoom[conn]
		mutexForClientRoom.RUnlock()

		if currentRoom == nil {
    		conn.Write([]byte("❌ Ошибка: вы не находитесь в комнате\n"))
    		return
		}

		if currentRoom.Name == "lobby" {
			conn.Write([]byte("❌ Вы не можете выйти из комнаты 'lobby'\n"))
			return
		}

		lobby := rooms["lobby"]
		currentRoom.RemoveClient(conn)
		currentRoom.sendMessageForRoom(conn, "❗ [" + name + "] покинул комнату " + currentRoom.Name)

		lobby.AddClient(conn, name)

		mutexForClientRoom.Lock()
		clientRoom[conn] = lobby
		mutexForClientRoom.Unlock()

		conn.Write([]byte("✅ Вы успешно вернулись в комнату lobby!\n"))
		lobby.sendMessageForRoom(conn, "✋ " + name + " вернулся в общий чат!")
	case "/rooms":
		mutexForRooms.RLock()
		defer mutexForRooms.RUnlock()

		if len(rooms) == 0 {
			conn.Write([]byte("❌ нет доступных комнат!\n"))
			return
		}

		totalRooms := ""
		count := 0
		for _, room := range rooms {
			count++
			lock := "🔓"
			if room.HavePassword() {
				lock = "🔒"
			} 
			totalRooms += fmt.Sprintf("   %d) %s %s (%d/%d человек)\n", count, lock, room.Name, room.CheckCountPeople(), globalConfig.MaxUserInRoom)
		}
		conn.Write([]byte(totalRooms))
	
	case "/help":
		result := ""
		trueCommand := []string{
			"/create <название> [пароль] - создать комнату",
			"/join <название> [пароль] - присоединиться к комнате",
			"/leave - покинуть текущую комнату",
			"/rooms - вывод списка доступных команд",
			"/exit - покинуть чат"}
		
		for _, val := range trueCommand {
			result += fmt.Sprintf("%s\n", val)
		}

		conn.Write([]byte(result))

	default:
		conn.Write([]byte("❌ неизвестная команда! (вызовите /help для вывода списка доступных команд)\n"))
	}
	
}

func splitCommand(s string) []string {

	var parts []string
	current := ""

	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	return parts

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