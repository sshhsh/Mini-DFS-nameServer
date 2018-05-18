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
)

var dataServer [4]string
var dataServerStatus [4]int

const RUNNING int = 1
const NONE int = 0
const ERROR int = 4
const RECOVERING int = 7
const BUFFLENGTH int = 2048 * 1024

func upload(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	//TODO

	w.Header().Add("Access-Control-Allow-Origin", "*")

	if r.Method == "POST" {
		fmt.Println(r.Header.Get("Content-Type"))

		file, handle, err := r.FormFile("file")
		if err != nil {
			fmt.Println(err)
		}

		chunkNum := int(handle.Size / int64(BUFFLENGTH))
		if handle.Size%int64(BUFFLENGTH) > 0 {
			chunkNum ++
		}

		newPath := r.PostFormValue("path")
		newMyFile := newFile(newPath, handle.Filename, true, chunkNum)

		reader := bufio.NewReader(file)

		i := 0
		offset := rand.Intn(4)
		var buff = make([]byte, BUFFLENGTH)
		for {
			n, err := reader.Read(buff)

			index := (i + offset) % 4

			//split to chunks
			newMyChunk := newChunk(index)
			newMyFile.chunks[i] = newMyChunk

			for j := 0; j < 4; j++ {
				if j != index {
					var data = make([]byte, n)
					copy(data, buff)
					go send(data, dataServer[j], newMyChunk.id.String())
				}
			}

			i++
			if err != nil && err != io.EOF {
				panic(err)
			}
			if n == 0 {
				break
			}
		}
		fmt.Printf("chunks: %d real: %d", chunkNum, i)

		defer file.Close()
		fmt.Println("upload success")
	}
}

func send(data []byte, target string, id string) {
	resp, err := http.Post("http://"+target+":8080", "", bytes.NewReader(data))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}
	//TODO
	fmt.Println(string(body))
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
	root = newFile("", "", false, 0)

	http.HandleFunc("/upload", upload)
	http.HandleFunc("/download", download)
	http.HandleFunc("/register", register)
	http.HandleFunc("/status", status)
	fmt.Println("Name server is running.")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
