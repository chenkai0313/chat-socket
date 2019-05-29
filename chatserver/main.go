package main

import (
	"net"
	"fmt"
	"log"
	"encoding/json"
	"github.com/satori/go.uuid"
	"time"
	"chat-socket/protocol"
	"strconv"
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

//组用户
type User struct {
	Name   string
	RoomId int
	UserId string
	Con    net.Conn
}

type Users struct {
	User []User
}

type Msg struct {
	Content     string
	Con         net.Conn
	Msg_type    int //1登陆消息 2内容消息 3系统消息
	SendName    string
	ContentType int //1 正常内容 2指令内容
}

var NeedSendMsgs = make(map[string]chan Msg) //消息组
var RoomManagers = make(map[int][]User)      //房间组
var UserManagers = make(map[string][]User)   //用户组
var Close = make(chan net.Conn)              //需要关闭的用户的连接

func main() {
	/**
	建立监听链接
	*/
	ln, err := net.Listen("tcp", "127.0.0.1:6011")
	if err != nil {
		panic(err)
	}
	go closeUserConnect()

	for {
		fmt.Println("wait connect...")
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal("get client connection error: ", err)
		}
		go handleConnect(conn)
	}
}

//处理链接
func handleConnect(conn net.Conn) {

	for {
		data := make([]byte, 1024)

		//可读缓存取
		readerChannel := make(chan []byte, 1024)
		//实际缓存区
		remainBuffer := make([]byte, 0)

		c, err := conn.Read(data)
		if err != nil {
			return
		}

		//解析自定义协议的数据
		remainBuffer = protocol.NewDefaultPacket(append(remainBuffer, data[:c]...)).UnPacket(readerChannel)

		//处理读取的数据
		go func(reader chan []byte) {
			for {

				packageData := <-reader

				receiveSocketContent := SocketContent{}

				unmarshalErr := json.Unmarshal(packageData, &receiveSocketContent)

				if unmarshalErr != nil {
					fmt.Println(unmarshalErr.Error())
				}

				//登陆
				if receiveSocketContent.Status == 1 && receiveSocketContent.ContentType == 1 {
					loginClinet(receiveSocketContent, conn)
				}

				//登陆中
				if receiveSocketContent.Status == 2 && receiveSocketContent.ContentType == 1 {
					chat(receiveSocketContent)
				}

				//获取获取指令内容
				if receiveSocketContent.Status == 2 && receiveSocketContent.ContentType == 2 {
					commandMsg(receiveSocketContent, conn)
				}
			}
		}(readerChannel)

	}
}

func commandMsg(receiveContent SocketContent, conn net.Conn) {

	timeTemplate1 := "2006-01-02 15:04:05"                            //常规类型
	time_str := time.Unix(time.Now().Unix(), 0).Format(timeTemplate1) //输出：2019-01-08 13:50:30
	fmt.Printf("【%v】房间名【%v】的人数【%v】\n", time_str, receiveContent.RoomId, len(RoomManagers[receiveContent.RoomId]))
	fmt.Printf("【%v】当前共有【%v】个房间 \n", time_str, len(RoomManagers))
	fmt.Printf("【%v】当前共有【%v】个用户 \n", time_str, len(UserManagers))

	//获取登陆用户总数
	if receiveContent.Content == "getLoginCounts" {
		//返回给当前用户系统消息
		var sys_msg Msg
		sys_msg.Con = conn
		sys_msg.Content = "当前登陆的人的总数:" + strconv.Itoa(len(UserManagers))
		sys_msg.Msg_type = 2
		sys_msg.SendName = "系统消息"
		sys_msg.ContentType = 2

		NeedSendMsgs[receiveContent.Uid] <- sys_msg
	}

	//获取房间内的用户总数
	if receiveContent.Content == "getRoomLoginCounts" {
		var sys_msg Msg
		sys_msg.Con = conn
		sys_msg.Content = "当前房间号[" + strconv.Itoa(receiveContent.RoomId) + "]内的人的总数:" + strconv.Itoa(len(RoomManagers[receiveContent.RoomId]))
		sys_msg.Msg_type = 2
		sys_msg.SendName = "系统消息"
		sys_msg.ContentType = 2

		NeedSendMsgs[receiveContent.Uid] <- sys_msg
	}

	//退出程序 回收数据
	if receiveContent.Content == "exit" {

		for i := 0; i < len(RoomManagers[receiveContent.RoomId]); i++ {
			RoomManagers[receiveContent.RoomId] = append(RoomManagers[receiveContent.RoomId][:i], RoomManagers[receiveContent.RoomId][i+1:]...)
		}

		delete(UserManagers, receiveContent.Uid)
		delete(NeedSendMsgs, receiveContent.Uid)

		var err error
		var send_msg SysMsg
		send_msg.MsgType = 2
		send_msg.MsgContent = "bye bye"
		send_msg.MgsUserId = receiveContent.Uid
		send_msg.MgsUserName = "系统消息"
		send_msg.ContentType = 4

		jsonContent, errs := json.Marshal(send_msg)
		if errs != nil {
			fmt.Println(errs.Error())
		}

		fmt.Printf("json len:[%v] \n", len(jsonContent))

		dataPackage := protocol.NewDefaultPacket([]byte(jsonContent)).Packet()

		_, err = conn.Write(dataPackage)
		if err != nil {
			Close <- conn
		}

		Close <- conn //断开连接
	}
}



func chat(receiveContent SocketContent) {
	//和房间内人聊天
	if receiveContent.SendUid == 0 {
		for _, v := range RoomManagers[receiveContent.RoomId] {
			var msg Msg
			msg.Con = v.Con
			msg.Content = receiveContent.Content
			msg.Msg_type = 2
			msg.SendName = receiveContent.Name
			msg.ContentType = 1

			NeedSendMsgs[v.UserId] <- msg
		}
	}
}

//判断用户名称是否重复
func userNameExist(receiveContent SocketContent, conn net.Conn) bool {

	for _, v := range UserManagers {
		for _, s := range v {
			if s.Name == receiveContent.Name {
				var err error
				var send_msg SysMsg
				send_msg.MsgType = 2
				send_msg.MsgContent = "用户名重复"
				send_msg.MgsUserId = ""
				send_msg.MgsUserName = receiveContent.Name
				send_msg.ContentType = 4

				jsonContent, errs := json.Marshal(send_msg)
				if errs != nil {
					fmt.Println(errs.Error())
				}

				fmt.Printf("json len:[%v] \n", len(jsonContent))

				//NewDefaultPacket
				dataPackage := protocol.NewDefaultPacket([]byte(jsonContent)).Packet()

				_, err = conn.Write(dataPackage)
				if err != nil {
					Close <- conn
				}

				return true
			}
		}
	}
	return false
}

//关闭用户连接
func closeUserConnect() {
	for {
		conn := <-Close
		conn.Close()
	}
}

//用户登陆 分配uid 创建用户的通道
func loginClinet(receiveContent SocketContent, conn net.Conn) {

	//判断用户名称是否重复
	Exist := userNameExist(receiveContent, conn)

	if Exist {
		return
	}

	//生成唯一uid
	uid, err := uuid.NewV4()
	if err != nil {
		fmt.Printf("Something went wrong: %s", err)
		return
	}
	uuid := uid.String()

	var user User
	user.Name = receiveContent.Name
	user.RoomId = receiveContent.RoomId
	user.UserId = uuid
	user.Con = conn

	timeTemplate1 := "2006-01-02 15:04:05"                            //常规类型
	time_str := time.Unix(time.Now().Unix(), 0).Format(timeTemplate1) //输出：2019-01-08 13:50:30

	//将用户的加入房间组
	RoomManagers[receiveContent.RoomId] = append(RoomManagers[receiveContent.RoomId], user)
	//将用户加入用户组
	UserManagers[uuid] = append(RoomManagers[receiveContent.RoomId], user)
	//用户创建信息通道
	NeedSendMsgs[uuid] = make(chan Msg, 50)

	fmt.Printf("【%v】房间名【%v】的人数【%v】\n", time_str, receiveContent.RoomId, len(RoomManagers[receiveContent.RoomId]))
	fmt.Printf("【%v】当前共有【%v】个房间 \n", time_str, len(RoomManagers))
	fmt.Printf("【%v】当前共有【%v】个用户 \n", time_str, len(UserManagers))

	//返回给当前用户系统消息
	var sys_msg Msg
	sys_msg.Con = conn
	sys_msg.Content = "系统消息"
	sys_msg.Msg_type = 3
	sys_msg.SendName = "系统消息"
	sys_msg.ContentType = 1
	NeedSendMsgs[uuid] <- sys_msg

	//用户第一次登陆通知所有房间内的人
	for _, v := range RoomManagers[receiveContent.RoomId] {
		welcome := "欢迎" + receiveContent.Name + "加入"
		var msg Msg
		msg.Con = v.Con
		msg.Content = welcome
		msg.Msg_type = 1
		msg.SendName = "系统"
		msg.ContentType = 1

		NeedSendMsgs[v.UserId] <- msg
	}

	//创建当前用户的协程读取信息
	go sendMsg(uuid)
}

//内容消息
type SysMsg struct {
	MsgType     int
	MsgContent  string
	MgsUserId   string
	MgsUserName string
	ContentType int //1 正常内容 2指令内容 4特殊消息 需要退出程序
}

//发送消息
func sendMsg(uuid string) {
	var err error
	for {
		talkContent := <-NeedSendMsgs[uuid]

		if talkContent.Msg_type == 1 || talkContent.Msg_type == 2 || talkContent.Msg_type == 3 {
			var sysMsg SysMsg
			sysMsg.MsgType = talkContent.Msg_type
			sysMsg.MgsUserId = uuid
			sysMsg.MsgContent = talkContent.Content
			sysMsg.ContentType = talkContent.ContentType

			sysMsg.MgsUserName = talkContent.SendName

			fmt.Println("need send")
			fmt.Println(sysMsg)

			jsonContent, errs := json.Marshal(sysMsg)
			if errs != nil {
				fmt.Println(errs.Error())
			}

			fmt.Printf("json len:[%v] \n", len(jsonContent))

			//NewDefaultPacket
			dataPackage := protocol.NewDefaultPacket([]byte(jsonContent)).Packet()

			_, err = talkContent.Con.Write(dataPackage)
			if err != nil {
				//closed <- true
			}
		}
	}
}
