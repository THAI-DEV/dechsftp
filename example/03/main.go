package main

import (
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

	dechsftp.DeleteAllInDir(client, remoteDir, true, true)
}
