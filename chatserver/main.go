package main

import (
	"net"
	"fmt"
	"log"
	"encoding/json"
	"github.com/satori/go.uuid"
	"time"
	"strconv"
	"math/rand"
	"strings"
	"chat-socket/protocol"
)

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
		go handleConnect(conn)
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

func handleConnect(conn net.Conn)  {
	////声明一个临时缓冲区，用来存储被截断的数据
	//tmpBuffer := make([]byte, 0)
	//buffer := make([]byte, 1024)
	//
	////声明一个管道用于接收解包的数据
	//
	//readerChannel := make(chan []byte, 16)
	//for {
	//	n, err := conn.Read(buffer)
	//	if err != nil {
	//		//log(conn.RemoteAddr().String(), " connection error: ", err)
	//		return
	//	}
	//
	//	tmpBuffer = protocol.Unpack(append(tmpBuffer, buffer[:n]...), readerChannel)
	//}

	for {
		data := make([]byte, 1024)
		c, err := conn.Read(data)
		if err != nil {
			//closed <- true  //这样会阻塞 | 后面取closed的for循环，没有执行到。
			return
		}

		//content:=string(data)
		//fmt.Println(content)
		//获取内容
		receiveContent := SocketContent{}
		unmarshalErr := json.Unmarshal(data[:c], &receiveContent)
		if unmarshalErr != nil {
			fmt.Println(unmarshalErr.Error())
		}

		//登陆
		if receiveContent.Status == 1{
			loginClinet(receiveContent,conn)
		}
	}
	return
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
	Content string
	Con net.Conn
	Msg_type int //1登陆消息 2内容消息 3系统消息
}

var NeedSendMsgs = make(map[string]chan Msg)    //消息组
var RoomManagers = make(map[int][]User) //房间组
var UserManagers = make(map[string][]User) //用户组


//用户登陆 分配uid 创建用户的通道
func loginClinet(receiveContent SocketContent,conn net.Conn)  {
	//生成唯一uid
	uid, err := uuid.NewV4()
	if err != nil {
		fmt.Printf("Something went wrong: %s", err)
		return
	}
	uuid:=uid.String()

	var user User
	user.Name=receiveContent.Name
	user.RoomId=receiveContent.RoomId
	user.UserId=uuid
	user.Con=conn

	timeTemplate1 := "2006-01-02 15:04:05" //常规类型
	time_str:=time.Unix(time.Now().Unix(), 0).Format(timeTemplate1) //输出：2019-01-08 13:50:30

	//将用户的加入房间组
	RoomManagers[receiveContent.RoomId]=append(RoomManagers[receiveContent.RoomId],user)
	//将用户加入用户组
	UserManagers[uuid]=append(RoomManagers[receiveContent.RoomId],user)
	//用户创建信息通道
	NeedSendMsgs[uuid]=make(chan Msg, 50)

	fmt.Printf("【%v】房间名【%v】的人数【%v】\n",time_str,receiveContent.RoomId,len(RoomManagers[receiveContent.RoomId]))
	fmt.Printf("【%v】当前共有【%v】个房间 \n",time_str,len(RoomManagers))
	fmt.Printf("【%v】当前共有【%v】个用户 \n",time_str,len(UserManagers))

	//返回给当前用户系统消息
	var sys_msg Msg
	sys_msg.Con=conn
	sys_msg.Content = "系统消息"
	sys_msg.Msg_type=3
	NeedSendMsgs[uuid] <-sys_msg


	//用户第一次登陆通知所有房间内的人
	for _,v:=range RoomManagers[receiveContent.RoomId]  {
		welcome := "欢迎" + receiveContent.Name + "加入"
		var msg Msg
		msg.Con=v.Con
		msg.Content=welcome
		msg.Msg_type=1
		NeedSendMsgs[v.UserId] <- msg
	}



	//创建当前用户的协程读取信息
	go sendMsg(uuid)
}

//内容消息
type SysMsg struct {
	MsgType int
	MsgContent string
	MgsUserId string
	MgsUserName string
}


//发送消息
func sendMsg(uuid string)  {
	var err error
	for {
		talkContent := <- NeedSendMsgs[uuid]
		if talkContent.Msg_type == 1 || talkContent.Msg_type == 2 || talkContent.Msg_type == 3 {
			var sysMsg SysMsg
			sysMsg.MsgType =talkContent.Msg_type
			sysMsg.MgsUserId=uuid
			sysMsg.MsgContent=talkContent.Content

			userName :=""
			for _,v:=range UserManagers[uuid] {
				userName=v.Name
			}

			sysMsg.MgsUserName=userName

			fmt.Println("need send")
			fmt.Println(sysMsg)


			jsonContent, errs := json.Marshal(sysMsg)
			if errs != nil {
				fmt.Println(errs.Error())
			}

			fmt.Printf("json len:[%v] \n",len(jsonContent))

			//NewDefaultPacket
			dataPackage := protocol.NewDefaultPacket([]byte(jsonContent)).Packet()

			_, err = talkContent.Con.Write(dataPackage)
			if err != nil {
				//closed <- true
			}
		}

	}
}


/**
*生成随机字符
**/
func RandString(length int) string {
	rand.Seed(time.Now().UnixNano())
	rs := make([]string, length)
	for start := 0; start < length; start++ {
		t := rand.Intn(3)
		if t == 0 {
			rs = append(rs, strconv.Itoa(rand.Intn(10)))
		} else if t == 1 {
			rs = append(rs, string(rand.Intn(26)+65))
		} else {
			rs = append(rs, string(rand.Intn(26)+97))
		}
	}
	return strings.Join(rs, "")
}












