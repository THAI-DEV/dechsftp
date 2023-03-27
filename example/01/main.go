package main

import (
	"fmt"
	"os"

	"github.com/THAI-DEV/dechsftp"
)

var host, port, username, password string

func init() {
	host = os.Getenv("HOST")
	port = os.Getenv("PORT")
	username = os.Getenv("USERNAME")
	password = os.Getenv("PASSWORD")
}

func main() {
	conn, err := dechsftp.NewConnection(host, port, username, password)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client, err := dechsftp.NewClient(conn)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	remoteDir := "/root/DECH/Test/a"

	list, err := dechsftp.GetDirNameAllLevel(client, remoteDir)
	if err != nil {
		fmt.Println(err)
	}

	list = dechsftp.ComputeAndOrderDirNameListByLevel(list, true)

	for _, v := range list {
		fmt.Println(v)
	}

	fmt.Println("-----------------------------")
	ss := dechsftp.Walk(client, remoteDir, true, true, true, false)
	for i, v := range ss {
		fmt.Println(i+1, v.Name, v.ModTime, v.ModTime, v.Level)
	}
}
