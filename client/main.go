//package main
//
//import "fmt"
//
//func main(){
//	for  {
//		var content string
//		fmt.Println("输入：")
//		fmt.Scanln(&content)
//		fmt.Printf("当前输入的是：%s \n",content)
//	}
//}

package main

import (
	"fmt"
	"math/rand"
	"net"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
)

type SendContent struct {
	Name string
	RoomId int
	Content string
	IsLogin int //1 登陆 2保持 3退出
	Uid int
}


func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:6011")
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	var name string
	var uid int
	fmt.Println("输入用户名：")
	fmt.Scanln(&name)

	fmt.Println("输入uid：")
	fmt.Scanln(&uid)

	var sendContent SendContent
	sendContent.Name=name
	sendContent.RoomId =1
	sendContent.Content="join"
	sendContent.IsLogin=1
	sendContent.Uid=uid

	jsons, errs := json.Marshal(sendContent)
	if errs != nil {
		fmt.Println(errs.Error())
	}
		_, err = conn.Write(jsons)
		if err != nil {
			fmt.Println("write to server error")
			return
		}

	go writeFromServer(conn)

	go cleanup()



	for {
		var talkContent string

		//fmt.Println("输入内容：")
		fmt.Scanln(&talkContent)

		var sendContent2 SendContent
		sendContent2.Name=name
		sendContent2.RoomId =1
		sendContent2.Content=talkContent
		sendContent2.IsLogin=2
		sendContent2.Uid=uid

		jsons2, errs := json.Marshal(sendContent2)
		if errs != nil {
			fmt.Println(errs.Error())
		}

		_, err = conn.Write(jsons2)
		if err != nil {
			fmt.Println("write to server error")
			return
		}

	}
}

func cleanup() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	/* 可以在下面定义一些资源回收，代码清理工作 */
	fmt.Println("exit...")
	os.Exit(3)
}

func writeFromServer(conn net.Conn) {
	defer conn.Close()
	for {
		data := make([]byte, 1024)
		c, err := conn.Read(data)
		if err != nil {
			fmt.Println("rand", rand.Intn(10), "have no server write", err)
			return
		}
		fmt.Println("收到的内容：")
		fmt.Println(string(data[0:c]) + "\n ")
	}
}













