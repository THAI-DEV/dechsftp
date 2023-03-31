package main

import (
	"fmt"
	"os"
	"strings"
	"time"

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

	remoteDir := "/root/DECH/Test/t"

	list := dechsftp.Walk(client, remoteDir, false)

	dirListExcludeRemoteDir := dechsftp.FilterFileInfoList(list, remoteDir, true, false, false, false)
	showList(dirListExcludeRemoteDir, "Dir List Exclude Remote Dir")

	filterList := filterIsBeforeList(dirListExcludeRemoteDir, remoteDir, 2) //2 day
	showList(filterList, "Dir List")
}

func showList(list []dechsftp.FileInfo, msg string) {
	fmt.Println()
	fmt.Println("-----------------------------", msg, "-----------------------------")
	for i, v := range list {
		fmt.Println(i+1, ":", v.Name, ":", v.ModTime, ":", v.ModTime, ":", v.Mode, ":", v.Level)
	}
}

func filterIsBeforeList(list []dechsftp.FileInfo, remoteDir string, day int) []dechsftp.FileInfo {
	result := []dechsftp.FileInfo{}
	for _, fileInfo := range list {
		folderName := fileInfo.Name
		folderName = strings.ReplaceAll(folderName, remoteDir+"/", "")
		if isBefore(folderName, day) {
			result = append(result, fileInfo)
		}
		// fmt.Println(folderName, isBefore(folderName))
	}

	return result
}

func isBefore(chkStrTime string, day int) bool {
	dayDuration := time.Duration(day)

	chkTime, err := time.Parse("2006-01-02", chkStrTime)
	if err != nil {
		fmt.Println(err)
	}

	currTime := time.Now().Format("2006-01-02")
	targetTime, _ := time.Parse("2006-01-02", currTime)
	targetTime = targetTime.Add(24 * time.Hour * dayDuration * -1)

	// fmt.Println("targetTime", targetTime.Format("2006-01-02"))

	return chkTime.Before(targetTime)
}
