package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateRoomNoPassword(t *testing.T) {
	rooms = make(map[string]*Room)
	currentRoom := CreateRoom("testRoom", "")
	assert.NotNil(t, currentRoom, "Комната не должна равняться nil")
	assert.Equal(t, "testRoom", currentRoom.Name, "Имя комнаты должно быть testRoom")
	assert.Empty(t, currentRoom.Password, "Пароль должен быть пустой!")

	savedRoom, ok := rooms["testRoom"]
	assert.True(t, ok, "Комната должна находиться в общей мапе комнат")
	assert.Same(t, currentRoom, savedRoom, "Комнаты не должны различаться")

}

func TestAddClientSuccses(t *testing.T) {
	rooms = make(map[string]*Room)
	clients = make(map[net.Conn]string)
	clientRoom = make(map[net.Conn]*Room)
	globalConfig = &Config{
		MaxUserInRoom: 2,
	}

	testNet, _ := net.Pipe()
	defer testNet.Close()
	TestRoom := CreateRoom("testRoom", "")
	TestRoom.AddClient(testNet, "Amir")
	assert.Equal(t, 1, TestRoom.CheckCountPeople(), "Количество людей в комнате должно равняться 1")
	assert.Equal(t, "Amir", TestRoom.Clients[testNet])
}

func TestAddClientInFullRoom(t *testing.T) {
	rooms = make(map[string]*Room)
	clients = make(map[net.Conn]string)
	clientRoom = make(map[net.Conn]*Room)
	globalConfig = &Config{
		MaxUserInRoom: 1,
	}

	testNet1, _ := net.Pipe()
	defer testNet1.Close()
	testNet2, _ := net.Pipe()
	defer testNet2.Close()

	testRoom := CreateRoom("testRoom", "")
	res1 := testRoom.AddClient(testNet1, "client1")
	assert.True(t, res1, "клиент должен быть успешно добавлен в комнату")
	assert.Equal(t, 1, testRoom.CheckCountPeople(), "кол-во клиентов должно быть в комнате = 1")

	res2 := testRoom.AddClient(testNet2, "client2")
	assert.False(t, res2, "второй клиент должен удариться oб ограничение")
	assert.Equal(t, 1, testRoom.CheckCountPeople(), "кол-во клиентов должно быть в комнате = 1")

}

func TestCreateRoomWithPassword(t *testing.T) {
	rooms = make(map[string]*Room)
	clients = make(map[net.Conn]string)
	clientRoom = make(map[net.Conn]*Room)

	testRoom := CreateRoom("testRoom", "123")
	assert.Equal(t, "testRoom", testRoom.Name, "Имя комнаты должно быть testRoom")
	assert.Equal(t, "123", testRoom.Password, "Пароль комнаты должен быть 123")


}

func TestRemoveClient(t *testing.T) {
	rooms = make(map[string]*Room)
	clients = make(map[net.Conn]string)
	clientRoom = make(map[net.Conn]*Room)
	globalConfig = &Config{
		MaxUserInRoom: 2,
	}

	testRoom := CreateRoom("testRoom", "")
	testClient, _ := net.Pipe()
	testRoom.AddClient(testClient, "client1")
	
	assert.Equal(t, 1, testRoom.CheckCountPeople(), "кол-во клиентов в комнате должно быть = 1")
	testRoom.RemoveClient(testClient)
	assert.Equal(t, 0, testRoom.CheckCountPeople(), "кол-во клиентов должно быть в комнате = 0")

}