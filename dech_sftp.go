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
	Mode    fs.FileMode
	Level   int
	File    string
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

	fileInfo := FileInfo{}

	files, err := client.ReadDir(remoteDir)
	if err != nil {
		// fmt.Printf("unable to list remote dir: %v\n", err)
		return result, err
	}

	for _, f := range files {
		fileInfo.Name = f.Name()
		fileInfo.ModTime = f.ModTime().Format("2006-01-02 15:04:05")
		fileInfo.Size = fmt.Sprintf("%12d", f.Size())
		fileInfo.IsDir = false
		fileInfo.Mode = f.Mode()
		fileInfo.Level = len(strings.Split(f.Name(), "/")) - 1
		fileInfo.File = f.Name()

		if f.IsDir() {
			fileInfo.IsDir = true
		}

		result = append(result, fileInfo)

	}

	return result, nil
}

func GetFileNameOneLevel(client *sftp.Client, remoteDir string) ([]string, error) {
	result := []string{}
	fileInfoList, err := ReadDirAndFileInfoOneLevel(client, remoteDir)
	if err != nil {
		return nil, err
	}

	for _, fileInfo := range fileInfoList {
		f := remoteDir + "/" + fileInfo.Name

		if !fileInfo.IsDir {
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

func DeleteAllInDir(client *sftp.Client, remoteDir string, isIncludeRemoteDir bool, isShowDebugMsg bool) error {
	fileInfoList := Walk(client, remoteDir, false)

	//* 1. Delete All File
	fileNameList := FilterFileInfoList(fileInfoList, remoteDir, false, true, false, false)

	for _, fileInfo := range fileNameList {
		err := client.Remove(fileInfo.Name)
		if err != nil {
			return err
		}

		if isShowDebugMsg {
			fmt.Println("File : " + fileInfo.Name)
		}
	}

	//* 2. Delete Sub Dir
	dirNameList := FilterFileInfoList(fileInfoList, remoteDir, true, false, false, false)

	dirNameList = OrderFileInfoList(dirNameList, true)

	for _, fileInfo := range dirNameList {
		err := client.Remove(fileInfo.Name)
		if err != nil {
			return err
		}

		if isShowDebugMsg {
			fmt.Println("Sub Dir : " + fileInfo.Name)
		}
	}

	//* 3. Delete Remote Dir
	if isIncludeRemoteDir {
		err := client.Remove(remoteDir)
		if err != nil {
			return err
		}

		if isShowDebugMsg {
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

func ChangeModeAllInDir(client *sftp.Client, remoteDir string, fileMode fs.FileMode, isIncludeRemoteDir bool, isShowDebugMsg bool) error {
	fileInfoList := Walk(client, remoteDir, false)

	//* 1. Change Remote Dir
	if isIncludeRemoteDir {
		err := client.Chmod(remoteDir, fileMode)
		if err != nil {
			return err
		}

		if isShowDebugMsg {
			fmt.Println("Remote Dir : " + remoteDir)
		}
	}

	//* 2.  Change Sub Dir
	dirNameList := FilterFileInfoList(fileInfoList, remoteDir, true, false, false, false)

	for _, fileInfo := range dirNameList {
		err := client.Chmod(fileInfo.Name, fileMode)
		if err != nil {
			return err
		}

		if isShowDebugMsg {
			fmt.Println("Sub Dir : " + fileInfo.Name)
		}
	}

	//* 3.  Change All File
	fileNameList := FilterFileInfoList(fileInfoList, remoteDir, false, true, false, false)

	for _, fileInfo := range fileNameList {
		err := client.Chmod(fileInfo.Name, fileMode)
		if err != nil {
			return err
		}

		if isShowDebugMsg {
			fmt.Println("File : " + fileInfo.Name)
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

func DownloadFile(client *sftp.Client, remoteFile, localFile string, isShowDebugMsg bool) (int64, error) {
	if isShowDebugMsg {
		fmt.Fprintf(os.Stdout, "Downloading [%s] to [%s] ...\n", remoteFile, localFile)
	}

	// Note: SFTP To Go doesn't support O_RDWR mode
	srcFile, err := client.OpenFile(remoteFile, (os.O_RDONLY))
	if err != nil {
		if isShowDebugMsg {
			fmt.Fprintf(os.Stderr, "Unable to open remote file: %v\n", err)
		}

		return -1, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localFile)
	if err != nil {
		if isShowDebugMsg {
			fmt.Fprintf(os.Stderr, "Unable to open local file: %v\n", err)
		}

		return -1, err
	}
	defer dstFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		if isShowDebugMsg {
			fmt.Fprintf(os.Stderr, "Unable to download remote file: %v\n", err)
		}

		os.Exit(1)
	}

	if isShowDebugMsg {
		fmt.Fprintf(os.Stdout, "%d bytes copied\n", bytes)
	}

	return bytes, err
}

func UploadFile(client *sftp.Client, localFile, remoteFile string, isShowDebugMsg bool) (int64, error) {
	if isShowDebugMsg {
		fmt.Fprintf(os.Stdout, "Uploading [%s] to [%s] ...\n", localFile, remoteFile)
	}

	srcFile, err := os.Open(localFile)
	if err != nil {
		if isShowDebugMsg {
			fmt.Fprintf(os.Stderr, "Unable to open local file: %v\n", err)
		}

		return -1, err
	}
	defer srcFile.Close()

	// Note: SFTP To Go doesn't support O_RDWR mode
	dstFile, err := client.OpenFile(remoteFile, (os.O_WRONLY | os.O_CREATE | os.O_TRUNC))
	if err != nil {
		if isShowDebugMsg {
			fmt.Fprintf(os.Stderr, "Unable to open remote file: %v\n", err)
		}

		return -1, err
	}
	defer dstFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		if isShowDebugMsg {
			fmt.Fprintf(os.Stderr, "Unable to upload local file: %v\n", err)
		}

		os.Exit(1)
	}

	if isShowDebugMsg {
		fmt.Fprintf(os.Stdout, "%d bytes copied\n", bytes)
	}

	return bytes, err
}

func Walk(client *sftp.Client, remoteDir string, isShowDebugMsg bool) []FileInfo {
	result := []FileInfo{}

	w := client.Walk(remoteDir)
	for w.Step() {
		if w.Err() != nil {
			continue
		}

		if isShowDebugMsg {
			fmt.Println(w.Path())
		}

		fInfo := FileInfo{
			Name:    w.Path(),
			ModTime: w.Stat().ModTime().String(),
			Size:    fmt.Sprintf("%12d", w.Stat().Size()),
			IsDir:   w.Stat().IsDir(),
			Mode:    w.Stat().Mode(),
			Level:   len(strings.Split(w.Path(), "/")) - 1,
			File:    w.Stat().Name(),
		}

		w.Stat().Sys()
		result = append(result, fInfo)
	}

	return result
}

func FilterFileInfoList(list []FileInfo, remoteDir string, isIncludeDir bool, isIncludeFile bool, isIncludeRemoteDir bool, isShowDebugMsg bool) []FileInfo {
	result := []FileInfo{}

	for _, fInfo := range list {
		if isShowDebugMsg {
			fmt.Println(fInfo.Name)
		}

		if isIncludeDir {
			if fInfo.IsDir {
				if !isIncludeRemoteDir {
					if fInfo.Name != remoteDir {
						result = append(result, fInfo)
					}
				}

				if isIncludeRemoteDir {
					result = append(result, fInfo)
				}
			}
		}

		if isIncludeFile {
			if !fInfo.IsDir {
				result = append(result, fInfo)
			}
		}
	}

	return result
}

// reverse : max -> min
func OrderFileInfoList(list []FileInfo, isReverse bool) []FileInfo {
	result := []FileInfo{}

	//Find Max Level
	maxLevel := -1
	for _, fileInfo := range list {
		if fileInfo.Level > maxLevel {
			maxLevel = fileInfo.Level
		}
	}

	if isReverse { // max -> min
		for i := maxLevel; i >= 1; i-- {
			for _, v := range list {
				if i == v.Level {
					result = append(result, v)
				}
			}
		}
	} else { // min -> max
		for i := 1; i <= maxLevel; i++ {
			for _, v := range list {
				if i == v.Level {
					result = append(result, v)
				}
			}
		}
	}

	return result
}
