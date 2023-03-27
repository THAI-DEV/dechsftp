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

	if host == "" {
		fmt.Println("**** You must set ENV ****")
	}
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

	oldRemoteFile := "/root/DECH/Test/2023-03-14/2023-03-14_06-00-01_UTL_backup.sql"
	newRemoteFile := "/root/DECH/Test/2023-03-14/xxxx.sql"

	dechsftp.RenameDirOrFile(client, oldRemoteFile, newRemoteFile)
}
