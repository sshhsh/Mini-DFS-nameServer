package main

import (
	"net/http"
	"log"
	"strings"
	"fmt"
	"bufio"
	"io"
	"math/rand"
	"bytes"
	"io/ioutil"
	"crypto/md5"
	"encoding/hex"
	"sync"
)

var dataServer [4]string
var dataServerStatus [4]int
var currentStatus bool

const RUNNING int = 1
const NONE int = 0
const ERROR int = 4
const RECOVERING int = 7
const BUFFLENGTH int = 2048 * 1024

func upload(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Add("Access-Control-Allow-Origin", "*")

	//check status
	currentStatus = cheackStatus()
	if !currentStatus {
		fmt.Println("Name Server is not ready")
		w.WriteHeader(500)
		return
	}

	if r.Method == "POST" {
		fmt.Println(r.Header.Get("Content-Type"))

		file, handle, err := r.FormFile("file")
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(500)
			return
		}
		defer file.Close()

		currentPath := r.PostFormValue("path")
		fmt.Printf("Saving %s into %s", handle.Filename, currentPath)
		if exists(currentPath, handle.Filename) {
			fmt.Println("file exists")
			w.WriteHeader(500)
			return
		}

		chunkNum := int(handle.Size / int64(BUFFLENGTH))
		if handle.Size%int64(BUFFLENGTH) > 0 {
			chunkNum ++
		}

		newChunks := make([]*Chunk, chunkNum)

		reader := bufio.NewReader(file)

		i := 0
		offset := rand.Intn(4)
		var buff = make([]byte, BUFFLENGTH)
		var waitgroup = new(sync.WaitGroup) //go routine number
		for {
			n, err := reader.Read(buff)

			if err != nil && err != io.EOF {
				panic(err)
				w.WriteHeader(500)
				return
			}
			if n == 0 {
				break
			}

			index := (i + offset) % 4

			//split to chunks
			newMyChunk := newChunk(index)
			newChunks[i] = newMyChunk

			for j := 0; j < 4; j++ {
				if j != index {
					var data = make([]byte, n)
					copy(data, buff)
					md5Ctx := md5.New()
					md5Ctx.Write(data)
					md5Result := md5Ctx.Sum(nil)
					waitgroup.Add(1)
					go send(data, j, newMyChunk.id.String(), hex.EncodeToString(md5Result), waitgroup)
				}
			}

			i++
		}

		waitgroup.Wait()
		if !cheackStatus() {
			fmt.Println("Maybe the last package went wrong")
			w.WriteHeader(500)
			return
		}

		newMyFile, err := newFile(currentPath, handle.Filename, true, chunkNum)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(500)
			return
		}
		newMyFile.chunks = newChunks

		fmt.Printf("chunks: %d real: %d", chunkNum, i)


		fmt.Println("upload success")
	}
}

func send(data []byte, target int, id string, md5 string, waitgroup *sync.WaitGroup) {
	defer waitgroup.Done()
	if !currentStatus {
		fmt.Printf("Something goes wrong when sending to %s.\n", dataServer[target])
		return
	}

	body := bytes.NewReader(data)
	request, err := http.NewRequest("POST", "http://"+dataServer[target]+":8080/upload", body)
	if err != nil {
		log.Println("http.NewRequest,[err=%s][url=%s]", err, dataServer[target])
		currentStatus = false
		dataServerStatus[target] = ERROR
		return
	}
	request.Header.Set("Connection", "Keep-Alive")
	request.Header.Set("filename", id)
	var resp *http.Response
	resp, err = http.DefaultClient.Do(request)
	if err != nil {
		log.Println("http.Do failed,[err=%s][url=%s]", err, dataServer[target])
		currentStatus = false
		dataServerStatus[target] = ERROR
		return
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("http.Do failed,[err=%s][url=%s]", err, dataServer[target])
	}

	if md5 != string(b) {
		fmt.Printf("MD5 checking failed")
		currentStatus = false
		dataServerStatus[target] = ERROR
		return
	}

	fmt.Printf("Writing to %s successful, MD5=%s\n", dataServer[target], md5)
}

func download(w http.ResponseWriter, r *http.Request) {
	//TODO

}

func register(w http.ResponseWriter, r *http.Request) {

	remoteAddr := strings.Split(r.RemoteAddr, ":")
	remoteIP := remoteAddr[0]

	for i := 0; i < 4; i++ {
		if dataServer[i] == "" {
			dataServer[i] = remoteIP
			dataServerStatus[i] = RUNNING
			w.Write([]byte("Success"))
			fmt.Printf("%s registered in place %d successfully!\n", remoteIP, i)
			break
		}
		if dataServerStatus[i] == ERROR {
			dataServer[i] = remoteIP
			//TODO recovery

			dataServerStatus[i] = RECOVERING
			w.Write([]byte("Success"))
			fmt.Printf("%s recovered with Place %d successfully!\n", remoteIP, i)
			break
		}
		if dataServer[i] == remoteIP {
			w.Write([]byte("Duplicated IP"))
			fmt.Printf("%s registered failed bacause of duplicated IP!\n", remoteIP)
			break
		}
	}

}

func status(w http.ResponseWriter, _ *http.Request) {
	for i := 0; i < 4; i++ {
		switch dataServerStatus[i] {
		case RUNNING:
			w.Write([]byte("RUNNING"))
		case RECOVERING:
			w.Write([]byte("RECOVERING"))
		case ERROR:
			w.Write([]byte("ERROR"))
		case NONE:
			w.Write([]byte("NONE"))
		}
		w.Write([]byte(dataServer[i]))
		w.Write([]byte("\n"))
	}
}

func cheackStatus() bool {
	for i := 0; i < 4; i++ {
		if dataServerStatus[i] != RUNNING {
			return false
		}
	}
	return true
}

func main() {
	var err error
	root, err = newFile("", "", false, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	http.HandleFunc("/upload", upload)
	http.HandleFunc("/download", download)
	http.HandleFunc("/register", register)
	http.HandleFunc("/status", status)
	fmt.Println("Name server is running.")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
