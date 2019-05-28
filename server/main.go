package main

import (
	"net"
	"fmt"
	"log"
	"encoding/json"
)

//接收的内容
type ReceiveContent struct {
	Name    string
	RoomId  int
	Content string
	IsLogin int //1 登陆 2保持 3退出
	Uid     int
}

//房间组
type UserGroup []User

//组用户
type User struct {
	Name   string
	RoomId int
	UserId int
	con    net.Conn
}

var userGroup UserGroup
var talkChan = make(map[int]chan string)

func main() {
	/**
	建立监听链接
	*/
	ln, err := net.Listen("tcp", "127.0.0.1:6011")
	if err != nil {
		panic(err)
	}

	for {
		fmt.Println("wait connect...")
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal("get client connection error: ", err)
		}

		go handleConnection(conn)
	}
}


func handleConnection(conn net.Conn) {
	//var err error
	var Uid int

	//var closed = make(chan bool)
	//
	//defer func() {
	//	fmt.Println("defer do : conn closed")
	//	conn.Close()
	//	fmt.Printf("delete userid [%v] from talkChan", Uid)
	//	delete(talkChan, Uid)
	//}()

	for {
		data := make([]byte, 1024)
		c, err := conn.Read(data)
		if err != nil {
			//closed <- true  //这样会阻塞 | 后面取closed的for循环，没有执行到。
			return
		}

		receiveContent := ReceiveContent{}
		unmarshalErr := json.Unmarshal(data[:c], &receiveContent)
		if unmarshalErr != nil {
			fmt.Println(unmarshalErr.Error())
		}
		var is_exit_room bool
		var is_first_join_room bool
		var first_str string
		is_first_join_room = false

		//判断用户是否是登陆
		if receiveContent.IsLogin == 1 {
			var user User
			user.Name = receiveContent.Name
			user.RoomId = receiveContent.RoomId

			user.UserId = receiveContent.Uid
			user.con = conn
			//判断用户是否在组中
			for _, v := range userGroup {
				if v.Name == receiveContent.Name && v.RoomId == receiveContent.RoomId {
					is_exit_room = true
				}
			}

			if !is_exit_room {
				userGroup = append(userGroup, user)
				is_first_join_room = true

				Uid = receiveContent.Uid

				talkChan[Uid] = make(chan string, 50)
				first_str = "欢迎" + receiveContent.Name + "加入"
			}
		}

		//if receiveContent.IsLogin == 3 {
		//	for _, v := range userGroup {
		//		if v.UserId == Uid{
		//
		//		}
		//	}
		//}


		fmt.Println(userGroup)

		for _, v := range userGroup {
			if is_first_join_room {
				talkChan[v.UserId] <- first_str
			}else {
				talkChan[v.UserId] <- receiveContent.Content
			}
		}

		for _, v := range userGroup {
		talkString := <-talkChan[v.UserId]
		_, err = v.con.Write([]byte(talkString))
			if err != nil {
				//closed <- true
			}
		}

	}

}
