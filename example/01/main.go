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

	list := dechsftp.Walk(client, remoteDir, false)

	dirAndFileList := dechsftp.FilterFileInfoList(list, remoteDir, true, true, true, false)
	showList(dirAndFileList, "All List")

	dirListIncludeRemoteDir := dechsftp.FilterFileInfoList(list, remoteDir, true, false, true, false)
	showList(dirListIncludeRemoteDir, "Dir List Include Remote Dir")

	dirListExcludeRemoteDir := dechsftp.FilterFileInfoList(list, remoteDir, true, false, false, false)
	showList(dirListExcludeRemoteDir, "Dir List Exclude Remote Dir")

	fileList := dechsftp.FilterFileInfoList(list, remoteDir, false, true, false, false)
	showList(fileList, "File List")

	orderFileList := dechsftp.OrderFileInfoList(fileList, true)
	showList(orderFileList, "Order File List")

	orderDirList := dechsftp.OrderFileInfoList(dirListIncludeRemoteDir, true)
	showList(orderDirList, "Order Dir List")
}

func showList(list []dechsftp.FileInfo, msg string) {
	fmt.Println()
	fmt.Println("-----------------------------", msg, "-----------------------------")
	for i, v := range list {
		fmt.Println(i+1, v.Name, v.ModTime, v.ModTime, v.Mode, v.Level)
	}
}
