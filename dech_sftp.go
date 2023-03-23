package dechsftp

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type FileInfo struct {
	Name    string
	ModTime string
	Size    string //byte
	IsDir   bool
}

func NewConnection(host string, port string, username, password string) (*ssh.Client, error) {
	// get host public key
	// hostKey := getHostKey(remote)
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// connect
	conn, err := ssh.Dial("tcp", host+":"+port, config)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return conn, nil
}

func NewClient(conn *ssh.Client) (*sftp.Client, error) {
	// create new SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	// defer client.Close()

	return client, nil
}

func ReadDirAndFileInfoOneLevel(client *sftp.Client, remoteDir string) ([]FileInfo, error) {
	var result []FileInfo

	info := FileInfo{}

	files, err := client.ReadDir(remoteDir)
	if err != nil {
		// fmt.Printf("unable to list remote dir: %v\n", err)
		return result, err
	}

	for _, f := range files {
		info.Name = f.Name()
		info.ModTime = f.ModTime().Format("2006-01-02 15:04:05")
		info.Size = fmt.Sprintf("%12d", f.Size())
		info.IsDir = false

		if f.IsDir() {
			info.IsDir = true
		}

		result = append(result, info)

	}

	return result, nil
}

// TODO
func ReadDirAndFileInfoAllLevel(client *sftp.Client, remoteDir string, isIncludeRemoteDir bool) ([]FileInfo, error) {
	dirList, err := GetDirNameAllLevel(client, remoteDir)
	if err != nil {
		return nil, err
	}

	result := []FileInfo{}
	for _, dirName := range dirList {
		fileInfoList, err := ReadDirAndFileInfoOneLevel(client, dirName)
		if err != nil {
			return nil, err
		}

		result = append(result, fileInfoList...)
	}

	if isIncludeRemoteDir {
		fileInfoList2, err := ReadDirAndFileInfoOneLevel(client, remoteDir)
		if err != nil {
			return nil, err
		}

		result = append(result, fileInfoList2...)
	}

	return result, nil
}

// * Exclude Remote Dir
func GetDirNameAllLevel(client *sftp.Client, remoteDir string) ([]string, error) {
	resultDir := []string{}
	fileInfoList, err := ReadDirAndFileInfoOneLevel(client, remoteDir)
	if err != nil {
		return nil, err
	}

	for _, v := range fileInfoList {
		f := remoteDir + "/" + v.Name
		if v.IsDir {
			resultDir = append(resultDir, f)
			subDir, _ := GetDirNameAllLevel(client, f)
			if len(subDir) > 0 {
				resultDir = append(resultDir, subDir...)
			}
		}
	}

	return resultDir, nil
}

// * Include File in Remote Dir
func GetFileNameAllLevel(client *sftp.Client, remoteDir string) ([]string, error) {
	dirList, err := GetDirNameAllLevel(client, remoteDir)
	if err != nil {
		return nil, err
	}

	dirList = append(dirList, remoteDir) // add init dir

	result := []string{}
	for _, v := range dirList {
		a, err := GetFileNameOneLevel(client, v)
		if err != nil {
			return nil, err
		}

		result = append(result, a...)
	}

	return result, nil
}

func GetFileNameOneLevel(client *sftp.Client, remoteDir string) ([]string, error) {
	result := []string{}
	fileInfoList, err := ReadDirAndFileInfoOneLevel(client, remoteDir)
	if err != nil {
		return nil, err
	}

	for _, v := range fileInfoList {
		f := remoteDir + "/" + v.Name

		if !v.IsDir {
			result = append(result, f)
		}
	}

	return result, nil
}

func CreateDir(client *sftp.Client, remoteDir string) error {
	err := client.Mkdir(remoteDir)
	if err != nil {
		return err
	}

	return nil
}

func DeleteAllInDir(client *sftp.Client, remoteDir string, isIncludeRemoteDir bool, isShowMsg bool) error {
	//* 1. Delete All File
	fileNameList, err := GetFileNameAllLevel(client, remoteDir)
	if err != nil {
		return err
	}

	for _, fileName := range fileNameList {
		err = client.Remove(fileName)
		if err != nil {
			return err
		}

		if isShowMsg {
			fmt.Println("File : " + fileName)
		}
	}

	//* 2. Delete Sub Dir
	dirNameList, err := GetDirNameAllLevel(client, remoteDir)
	if err != nil {
		return err
	}

	dirNameList = ComputeAndOrderDirNameListByLevel(dirNameList, true)

	for _, dirName := range dirNameList {
		err = client.Remove(dirName)
		if err != nil {
			return err
		}

		if isShowMsg {
			fmt.Println("Sub Dir : " + remoteDir)
		}
	}

	//* 3. Delete Remote Dir
	if isIncludeRemoteDir {
		err = client.Remove(remoteDir)
		if err != nil {
			return err
		}

		if isShowMsg {
			fmt.Println("Remote Dir : " + remoteDir)
		}
	}

	return nil
}

func DeleteEmptyDirOrFile(client *sftp.Client, remoteDirOrFile string) error {
	err := client.Remove(remoteDirOrFile)
	if err != nil {
		return err
	}

	return nil
}

func RenameDirOrFile(client *sftp.Client, oldRemoteDirOrFile string, newRemoteDirOrFile string) error {
	err := client.Rename(oldRemoteDirOrFile, newRemoteDirOrFile)
	if err != nil {
		return err
	}

	return nil
}

func ChangeModeAllInDir(client *sftp.Client, remoteDir string, fileMode fs.FileMode, isIncludeRemoteDir bool, isShowMsg bool) error {
	//* 1. Change Remote Dir
	if isIncludeRemoteDir {
		err := client.Chmod(remoteDir, fileMode)
		if err != nil {
			return err
		}

		if isShowMsg {
			fmt.Println("Remote Dir : " + remoteDir)
		}
	}

	//* 2.  Change Sub Dir
	dirNameList, err := GetDirNameAllLevel(client, remoteDir)
	if err != nil {
		return err
	}

	for _, dirName := range dirNameList {
		err = client.Chmod(dirName, fileMode)
		if err != nil {
			return err
		}

		if isShowMsg {
			fmt.Println("Sub Dir : " + dirName)
		}
	}

	//* 3.  Change All File
	fileNameList, err := GetFileNameAllLevel(client, remoteDir)
	if err != nil {
		return err
	}

	for _, fileName := range fileNameList {
		err = client.Chmod(fileName, fileMode)
		if err != nil {
			return err
		}

		if isShowMsg {
			fmt.Println("File : " + fileName)
		}
	}

	return nil
}

func ChangeModeDirOrFile(client *sftp.Client, remoteDirOrFile string, fileMode fs.FileMode) error {
	err := client.Chmod(remoteDirOrFile, fileMode)
	if err != nil {
		return err
	}

	return nil
}

func DownloadFile(client *sftp.Client, remoteFile, localFile string, isShowMsg bool) (int64, error) {
	if isShowMsg {
		fmt.Fprintf(os.Stdout, "Downloading [%s] to [%s] ...\n", remoteFile, localFile)
	}

	// Note: SFTP To Go doesn't support O_RDWR mode
	srcFile, err := client.OpenFile(remoteFile, (os.O_RDONLY))
	if err != nil {
		if isShowMsg {
			fmt.Fprintf(os.Stderr, "Unable to open remote file: %v\n", err)
		}

		return -1, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localFile)
	if err != nil {
		if isShowMsg {
			fmt.Fprintf(os.Stderr, "Unable to open local file: %v\n", err)
		}

		return -1, err
	}
	defer dstFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		if isShowMsg {
			fmt.Fprintf(os.Stderr, "Unable to download remote file: %v\n", err)
		}

		os.Exit(1)
	}

	if isShowMsg {
		fmt.Fprintf(os.Stdout, "%d bytes copied\n", bytes)
	}

	return bytes, err
}

func UploadFile(client *sftp.Client, localFile, remoteFile string, isShowMsg bool) (int64, error) {
	if isShowMsg {
		fmt.Fprintf(os.Stdout, "Uploading [%s] to [%s] ...\n", localFile, remoteFile)
	}

	srcFile, err := os.Open(localFile)
	if err != nil {
		if isShowMsg {
			fmt.Fprintf(os.Stderr, "Unable to open local file: %v\n", err)
		}

		return -1, err
	}
	defer srcFile.Close()

	// Note: SFTP To Go doesn't support O_RDWR mode
	dstFile, err := client.OpenFile(remoteFile, (os.O_WRONLY | os.O_CREATE | os.O_TRUNC))
	if err != nil {
		if isShowMsg {
			fmt.Fprintf(os.Stderr, "Unable to open remote file: %v\n", err)
		}

		return -1, err
	}
	defer dstFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		if isShowMsg {
			fmt.Fprintf(os.Stderr, "Unable to upload local file: %v\n", err)
		}

		os.Exit(1)
	}

	if isShowMsg {
		fmt.Fprintf(os.Stdout, "%d bytes copied\n", bytes)
	}

	return bytes, err
}

// reverse : max -> min
func ComputeAndOrderDirNameListByLevel(dirNameList []string, isReverse bool) []string {
	type fileAttrType struct {
		name  string
		level int
	}

	fileAttr := []fileAttrType{}
	maxLevel := -1

	for _, fileName := range dirNameList {
		a := strings.Split(fileName, "/")
		levelFileName := len(a)

		data := fileAttrType{
			name:  fileName,
			level: levelFileName,
		}

		fileAttr = append(fileAttr, data)

		if levelFileName > maxLevel {
			maxLevel = levelFileName
		}
	}

	result := []string{}
	if isReverse { // max -> min
		for i := maxLevel; i >= 1; i-- {
			for _, v := range fileAttr {
				if i == v.level {
					result = append(result, v.name)
				}
			}
		}
	} else { // min -> max
		for i := 1; i <= maxLevel; i++ {
			for _, v := range fileAttr {
				if i == v.level {
					result = append(result, v.name)
				}
			}
		}
	}

	return result
}

// func GetSepalateNameAllInDir(client *sftp.Client, remoteDir string) ([]string, []string, error) {
// 	resultFileName := []string{}
// 	resultDirName := []string{}
// 	fileInfoList, err := ReadDirAndFileInfoOneLevel(client, remoteDir)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	for _, v := range fileInfoList {
// 		f := remoteDir + "/" + v.Name
// 		if v.IsDir {
// 			resultDirName = append(resultDirName, f)
// 		} else {
// 			resultFileName = append(resultFileName, f)
// 		}

// 	}

// 	return resultDirName, resultFileName, nil
// }
