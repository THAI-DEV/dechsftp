package dechsftp

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"

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

func ReadDir(client *sftp.Client, remoteDir string) ([]FileInfo, error) {
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

func GetNameAllInDir(client *sftp.Client, remoteDir string) ([]string, error) {
	result := []string{}
	fileInfoList, err := ReadDir(client, remoteDir)
	if err != nil {
		return nil, err
	}

	for _, v := range fileInfoList {
		f := remoteDir + "/" + v.Name
		result = append(result, f)
	}

	return result, nil
}

func GetSepalateNameAllInDir(client *sftp.Client, remoteDir string) ([]string, []string, error) {
	resultFileName := []string{}
	resultDirName := []string{}
	fileInfoList, err := ReadDir(client, remoteDir)
	if err != nil {
		return nil, nil, err
	}

	for _, v := range fileInfoList {
		f := remoteDir + "/" + v.Name
		if v.IsDir {
			resultDirName = append(resultDirName, f)
		} else {
			resultFileName = append(resultFileName, f)
		}

	}

	return resultDirName, resultFileName, nil
}

func CreateDir(client *sftp.Client, remoteDir string) error {
	err := client.Mkdir(remoteDir)
	if err != nil {
		return err
	}

	return nil
}

func DeleteAllInDir(client *sftp.Client, remoteDir string, isShowMsg bool) error {
	fileNameList, err := GetNameAllInDir(client, remoteDir)
	if err != nil {
		return err
	}

	for _, remoteFile := range fileNameList {
		err = client.Remove(remoteFile)
		if err != nil {
			return err
		}

		if isShowMsg {
			fmt.Println("File : " + remoteFile)
		}
	}

	err = client.Remove(remoteDir)
	if err != nil {
		return err
	}

	if isShowMsg {
		fmt.Println("Dir : " + remoteDir)
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

func ChangeModeAllInDir(client *sftp.Client, remoteDir string, fileMode fs.FileMode, isShowMsg bool) error {
	fileNameList, err := GetNameAllInDir(client, remoteDir)
	if err != nil {
		return err
	}

	err = client.Chmod(remoteDir, fileMode)
	if err != nil {
		return err
	}

	if isShowMsg {
		fmt.Println("Dir : " + remoteDir)
	}

	for _, remoteFile := range fileNameList {
		err = client.Chmod(remoteFile, fileMode)
		if err != nil {
			return err
		}

		if isShowMsg {
			fmt.Println("File : " + remoteFile)
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

// Download file from sftp server
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

// Upload file to sftp server
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
