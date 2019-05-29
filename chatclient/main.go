package main

import (
	"fmt"
	"encoding/json"
	"net"
	"math/rand"
	"chat-socket/protocol"
	"strconv"
	"os"
	"os/signal"
)

type SocketContent struct {
	Name        string
	Uid         string //0为默认值
	RoomId      int
	Status      int //1 登陆 2登陆中 3退出
	Content     string
	SendUid     int //0为默认值
	ContentType int //1 正常内容 2指令内容
}

//定义需要发送的数据通道
var NeedSeed = make(chan SocketContent, 1024)
var name string
var uid string
var roomId int

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8089")
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	getHelp()

	firstInput()

	//第一次登陆
	login()

	go forcedLayout()
	//接收消息
	go receive(conn)
	//发送消息
	go send(conn)

	for {
		var talkContent string

		fmt.Scanln(&talkContent)
		if talkContent!=""{
			//判断内容
			switch talkContent {
			case "getLoginCounts":
				chat(talkContent,2) //获取所有登陆中的人的总数
			case "getRoomLoginCounts":
				chat(talkContent,2)//获取房间内的人的总数
			case "getUid":
				getUid() //获取用户id
			case "help":
				getHelp() //获取用户id
			case "exit":
				chat(talkContent,2)
			default:
				chat(talkContent,1)
			}
		}
	}
}

//捕捉ctrl+c/z 退出
func forcedLayout()  {
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for  range signalChan {
			//
			chat("exit",2)
			cleanupDone <- true
		}
	}()
	<-cleanupDone
}


func chat(talkContent string,ContentType int) {
	var socketContent SocketContent
	socketContent.Name = name
	socketContent.Uid = uid
	socketContent.RoomId = roomId
	socketContent.Status = 2
	socketContent.Content = talkContent
	socketContent.SendUid = 0
	socketContent.ContentType = ContentType

	NeedSeed <- socketContent
}

func getHelp() {
	var helps = [5]string{"---getLoginCounts  获取所有登陆中的人的总数", "---getRoomLoginCounts 获取房间内的人的总数", "---getUid 获取用户id", "---exit 退出程序","---help 获取更多帮助"}
	for _, v := range helps {
		fmt.Println(v)
	}
}

func firstInput() {
	var TempName string
	var TempRoomIdString string

	for {
		fmt.Println("输入用户名：")
		fmt.Scanln(&TempName)
		if TempName == ""{
			continue
		}
		name = TempName

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
	socketContent.ContentType = 1

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
	fmt.Printf("系统消息：name [%s] userid [%s] \n", name,uid)
}



type ReceiveSocketContent struct {
	MsgType     int //1登陆消息 2内容消息 3系统消息
	MsgContent  string
	MgsUserId   string
	MgsUserName string
	ContentType int //1 正常内容 2指令内容
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

				//内容消息展示
				if receiveSocketContent.ContentType == 1{

					//将用户的uid 赋值
					if receiveSocketContent.MsgType == 3 {
						uid = receiveSocketContent.MgsUserId
					}

					if receiveSocketContent.MsgType == 1 {
						fmt.Printf("系统消息：%s \n", receiveSocketContent.MsgContent)
					}

					//fmt.Println(receiveSocketContent)
					if receiveSocketContent.MsgType == 2 && receiveSocketContent.MgsUserName != name {
						fmt.Printf("【%s】:%s \n", receiveSocketContent.MgsUserName, receiveSocketContent.MsgContent)
					}

				}

				//指令消息展示
				if receiveSocketContent.ContentType == 2{
					fmt.Printf("【%s】:%s \n", receiveSocketContent.MgsUserName, receiveSocketContent.MsgContent)
				}
				//指令消息展示
				if receiveSocketContent.ContentType == 4{
					fmt.Printf("【%s】:%s \n", receiveSocketContent.MgsUserName, receiveSocketContent.MsgContent)
					os.Exit(3)
				}

			}
		}(readerChannel)
	}
}
