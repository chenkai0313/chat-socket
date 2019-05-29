package main

import (
	"fmt"
	"encoding/json"
	"net"
	"math/rand"
	"chat-socket/protocol"
	"strconv"
)

type SocketContent struct {
	Name    string
	Uid     string //0为默认值
	RoomId  int
	Status  int //1 登陆 2登陆中 3退出
	Content string
	SendUid int //0为默认值
}

//定义需要发送的数据通道
var NeedSeed = make(chan SocketContent, 1024)
var name string
var uid string
var roomId int

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:6011")
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	firstInput()

	//第一次登陆
	login()
	//接收消息
	go receive(conn)
	//发送消息
	go send(conn)

	for {
		var talkContent string

		fmt.Scanln(&talkContent)

		//fmt.Println("您：", talkContent)

		//判断内容
		switch talkContent {
		case "listFriends":
			listFriends() //获取所有登陆中的人
		case "listRoomFrineds":
			listRoomClients() //获取房间内的人
		case "getUid":
			getUid() //获取用户id
		case "help":
			getHelp() //获取用户id
		default:
			chatWithRoom(talkContent)
		}

	}
}

//和同一个房间内的人聊天
func chatWithRoom(talkContent string)  {
	var socketContent SocketContent
	socketContent.Name = name
	socketContent.Uid = uid
	socketContent.RoomId = roomId
	socketContent.Status = 2
	socketContent.Content = talkContent
	socketContent.SendUid = 0

	NeedSeed <- socketContent
}

func getHelp()  {
	var helps = [5]string{"---listFriends  获取所有登陆中的人", "---listRoomFrineds 获取房间内的人","---getUid 获取用户id","---help 获取更多帮助"}
	for _,v:=range helps {
		fmt.Println(v)
	}
}

func firstInput() {
	var TempName string
	fmt.Println("输入用户名：")
	fmt.Scanln(&TempName)

	var TempRoomIdString string

	for {
		var TempRoomIdInt int

		fmt.Println("输入加入的roomId：")
		fmt.Scanln(&TempRoomIdString)
		TempRoomIdInt, _ = strconv.Atoi(TempRoomIdString)

		if TempRoomIdInt > 0 {
			roomId = TempRoomIdInt
			break
		}
		fmt.Println("请输入有效值必须大于0的整数")
	}
	name = TempName

	fmt.Printf("输入的用户名为[%v],roomid为[%v] \n", name, roomId)
}

//登陆
func login() {
	var socketContent SocketContent
	socketContent.Name = name
	socketContent.Uid = "0"
	socketContent.RoomId = roomId
	socketContent.Status = 1
	socketContent.Content = "login"
	socketContent.SendUid = 0

	NeedSeed <- socketContent
}

//发送数据
func send(conn net.Conn) {
	defer conn.Close()
	for {

		send := <-NeedSeed

		jsonContent, errs := json.Marshal(send)
		if errs != nil {
			fmt.Println(errs.Error())
		}

		//添加数据协议
		dataPackage := protocol.NewDefaultPacket([]byte(jsonContent)).Packet()

		var err error
		_, err = conn.Write(dataPackage)
		if err != nil {
			fmt.Println("write to server error")
			return
		}
	}

}

func getUid() {
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
	MsgType     int //1登陆消息 2内容消息 3系统消息
	MsgContent  string
	MgsUserId   string
	MgsUserName string
}

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
		remainBuffer = protocol.NewDefaultPacket(append(remainBuffer, data[:c]...)).UnPacket(readerChannel)

		go func(reader chan []byte) {
			for {

				packageData := <-reader

				receiveSocketContent := ReceiveSocketContent{}

				unmarshalErr := json.Unmarshal(packageData, &receiveSocketContent)

				if unmarshalErr != nil {
					fmt.Println(unmarshalErr.Error())
				}

				//将用户的uid 赋值
				if receiveSocketContent.MsgType == 3 {
					uid = receiveSocketContent.MgsUserId
				}

				if receiveSocketContent.MsgType == 1 {
					fmt.Printf("系统消息：%s \n", receiveSocketContent.MsgContent)
				}

				//fmt.Println(receiveSocketContent)

				if receiveSocketContent.MsgType == 2 && receiveSocketContent.MgsUserName!=name{
					fmt.Printf("【%s】:%s \n", receiveSocketContent.MgsUserName, receiveSocketContent.MsgContent)
				}



			}
		}(readerChannel)
	}
}
