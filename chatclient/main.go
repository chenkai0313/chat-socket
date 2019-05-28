package main

import (
	"fmt"
	"encoding/json"
	"net"
	"math/rand"

	"chat-socket/protocol"
)

var name string
var uid string
var roomId int


func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:6011")
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	fmt.Println("输入用户名：")
	fmt.Scanln(&name)

	fmt.Println("输入加入的roomId：")
	fmt.Scanln(&roomId)

	fmt.Printf("输入的用户名为[%v],roomid为[%v] \n", name, roomId)

	//第一次登陆
	login(conn)
	//接收消息
	go receive(conn)


	for {
		var talkContent string

		fmt.Scanln(&talkContent)

		fmt.Println("您输入的内容：", talkContent)

		fmt.Println(123)
		fmt.Println(uid)

		//判断内容
		switch talkContent {
		case "listFriends":
			listFriends()
		case "listRoomFrineds":
			listRoomClients()
		}

	}

}

type SocketContent struct {
	Name    string
	Uid     int //0为默认值
	RoomId  int
	Status  int  //1 登陆 2登陆中 3退出
	Content string
	SendUid int //0为默认值
}


//登陆
func login(conn net.Conn) {
	var socketContent SocketContent
	socketContent.Name=name
	socketContent.Uid=0
	socketContent.RoomId =roomId
	socketContent.Status =1
	socketContent.Content ="loin"
	socketContent.SendUid = 0

	jsonContent, errs := json.Marshal(socketContent)
	if errs != nil {
		fmt.Println(errs.Error())
	}

	var err error
	_, err = conn.Write(jsonContent)
	if err != nil {
		fmt.Println("write to server error")
		return
	}

}

func getUid()  {
	fmt.Println("get uid")
}


//获取所有登陆中的人
func listFriends() {
	fmt.Println("list friends")
}

//获取房间内的人
func listRoomClients() {
	fmt.Println("list room clients")

}

//退出 ctrl+c 平滑退出
func layout() {
	fmt.Println("layout")
}

type ReceiveSocketContent struct {
	MsgType int //1登陆消息 2内容消息 3系统消息
	MsgContent string
	MgsUserId string
	MgsUserName string
}

//发送数据
//func send()


//显示收到的内容
func receive(conn net.Conn) {
	defer conn.Close()

	for {
		data := make([]byte, 1024)
		c, err := conn.Read(data)
		if err != nil {
			fmt.Println("rand", rand.Intn(10), "have no server write", err)
			return
		}

		//可读缓存取
		readerChannel := make(chan []byte, 1024)
		//实际缓存区
		remainBuffer := make([]byte, 0)

		//解析自定义的协议
		remainBuffer =   protocol.NewDefaultPacket(append(remainBuffer,data[:c]...)).UnPacket(readerChannel)

		go func(reader chan []byte) {
			for {

				packageData := <- reader

				receiveSocketContent := ReceiveSocketContent{}

				unmarshalErr := json.Unmarshal(packageData, &receiveSocketContent)

				if unmarshalErr != nil {
					fmt.Println(unmarshalErr.Error())
				}

				//将用户的uid 赋值
				if receiveSocketContent.MsgType == 3{
					uid = receiveSocketContent.MgsUserId
				}

				if receiveSocketContent.MsgType == 1 {
					fmt.Printf("系统消息：%s \n",receiveSocketContent.MsgContent)
				}
				if receiveSocketContent.MsgType == 2 {
					fmt.Printf("【%s】:%s",receiveSocketContent.MgsUserName,receiveSocketContent.MsgContent)
				}

			}
		}(readerChannel)
	}
}
