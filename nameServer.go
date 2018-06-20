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
	"encoding/json"
)

var dataServer [4]string
var dataServerStatus [4]int
var currentStatus bool

const RUNNING = 1
const NONE = 0
const ERROR = 4
const RECOVERING = 7
const BUFFLENGTH = 2048 * 1024

func upload(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Add("Access-Control-Allow-Origin", "*")

	//check status
	currentStatus = checkStatus()
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
		if !checkStatus() {
			fmt.Println("Maybe the last package went wrong")
			w.WriteHeader(500)
			return
		}

		newMyFile, err := newFile(currentPath, handle.Filename, true)
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

func newFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	currentPath := r.FormValue("path")
	newDir := r.FormValue("newPath")
	if newDir == "" {
		fmt.Println("no folder name")
		w.WriteHeader(500)
		return
	}
	_, err := newFile(currentPath, newDir, false)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	fmt.Println("create new folder success")
}

func send(data []byte, target int, id string, md5 string, waitgroup *sync.WaitGroup) {
	if waitgroup != nil {
		defer waitgroup.Done()
	}
	if !currentStatus && dataServerStatus[target] != RECOVERING {
		fmt.Printf("Something goes wrong when sending to %s.\n", dataServer[target])
		return
	}

	body := bytes.NewReader(data)
	request, err := http.NewRequest("POST", "http://"+dataServer[target]+":8080/upload", body)
	if err != nil {
		log.Printf("http.NewRequest,[err=%s][url=%s]", err, dataServer[target])
		currentStatus = false
		dataServerStatus[target] = ERROR
		return
	}
	request.Header.Set("Connection", "Keep-Alive")
	request.Header.Set("filename", id)
	var resp *http.Response
	resp, err = http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("http.Do failed,[err=%s][url=%s]", err, dataServer[target])
		currentStatus = false
		dataServerStatus[target] = ERROR
		return
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("http.Do failed,[err=%s][url=%s]", err, dataServer[target])
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
	w.Header().Add("Access-Control-Allow-Origin", "*")

	//check status
	currentStatus = checkStatus()
	if !currentStatus {
		fmt.Println("Name Server is not ready")
		w.WriteHeader(500)
		return
	}

	path := r.FormValue("path")
	host := r.RemoteAddr
	fmt.Printf("Sending %s to %s.\n", path, host)
	file := getFileFromPath(path)
	if file == nil || !file.isFile {
		fmt.Println("File don't exists")
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("content-disposition", "attachment; filename=\""+file.basename+"\"")

	for i, chunk := range file.chunks {
		res := receive(chunk.server[i%3], chunk.id.String())
		if res == nil {
			fmt.Printf("Receiving from %d failed.", chunk.server[i%3])
			dataServerStatus[chunk.server[i%3]] = ERROR
			w.WriteHeader(500)
			return
		}
		w.Write(res)
	}
}

func receive(target int, id string) []byte {
	request, err := http.NewRequest("GET", "http://"+dataServer[target]+":8080/download", nil)
	if err != nil {
		log.Printf("http.NewRequest,[err=%s][url=%s]", err, dataServer[target])
		currentStatus = false
		dataServerStatus[target] = ERROR
		return nil
	}
	request.Header.Set("filename", id)
	var resp *http.Response
	resp, err = http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("http.Do failed,[err=%s][url=%s]", err, dataServer[target])
		currentStatus = false
		dataServerStatus[target] = ERROR
		return nil
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("http.Do failed,[err=%s][url=%s]", err, dataServer[target])
	}
	return b
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
			dataServerStatus[i] = RECOVERING
			go recovery(i)
			w.Write([]byte("Success"))
			break
		}
		if dataServer[i] == remoteIP {
			w.Write([]byte("Duplicated IP"))
			fmt.Printf("%s registered failed bacause of duplicated IP!\n", remoteIP)
			break
		}
	}

}

func recovery(target int) {
	resp, err := http.Get("http://" + dataServer[target] + ":8080/echo")
	if err != nil {
		go recovery(target)
		return
	}
	resp.Body.Close()
	fmt.Printf("%s echo success\n", dataServer[target])
	queue := make([]*MyFile, 0)
	queue = append(queue, root) //push
	for {
		if len(queue) == 0 { //is empty
			break
		}
		currentDir := queue[0] //top
		for _, dir := range currentDir.files {
			if dir == currentDir {
				continue
			}
			if !dir.isFile {
				queue = append(queue, dir)
			} else {
				for _, ch := range dir.chunks {
					for _, server := range ch.server {
						if server == target {
							for _, server2 := range ch.server {
								if dataServerStatus[server2] == RUNNING {
									recoveryTo(server2, server, ch.id.String())
									break
								}
							}
							break
						}
					}
				}
			}
		}
		queue = queue[1:] //pop
	}

	fmt.Printf("%s recovered with Place %d successfully!\n", dataServer[target], target)
	dataServerStatus[target] = RUNNING
}

func recoveryTo(from int, to int, id string) {
	fmt.Printf("%s send to %s\n", dataServer[from], dataServer[to])
	tmp := receive(from, id)
	md5Ctx := md5.New()
	md5Ctx.Write(tmp)
	md5Result := md5Ctx.Sum(nil)
	send(tmp, to, id, hex.EncodeToString(md5Result), nil)
}

func status(w http.ResponseWriter, _ *http.Request) {
	currentStatus = checkStatus()
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

func checkStatus() bool {
	tmp := true
	for i := 0; i < 4; i++ {
		resp, err := http.Get("http://" + dataServer[i] + ":8080/echo")
		if err != nil {
			tmp = false
			dataServerStatus[i] = ERROR
		} else {
			resp.Body.Close()
			dataServerStatus[i] = RUNNING
		}
	}
	return tmp
}

func list(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	path := r.FormValue("path")
	dir := getFileFromPath(path)
	if dir == nil {
		fmt.Println("No such directory.")
		w.WriteHeader(500)
		return
	}
	if dir.isFile {
		fmt.Println("Not a directory.")
		w.WriteHeader(500)
		return
	}
	s := make([]map[string]string, len(dir.files)-1)
	for i, f := range dir.files {
		if i == 0 {
			continue
		}
		s[i-1] = make(map[string]string)
		if f.isFile {
			s[i-1]["type"] = "file"
		} else {
			s[i-1]["type"] = "dir"
		}
		if path == "" {
			s[i-1]["path"] = f.basename
		} else {
			s[i-1]["path"] = f.path + "/" + f.basename
		}

		s[i-1]["basename"] = f.basename
		s[i-1]["extension"] = f.extension
		s[i-1]["filename"] = f.filename
	}
	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println("error:", err)
	}
	w.Write(b)
}

func main() {
	root, _ = newFile("", "", false)
	_, _ = newFile("", "someDir", false)
	_, _ = newFile("", "otherDir", false)
	_, _ = newFile("otherDir", "anotherDir", false)
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/download", download)
	http.HandleFunc("/register", register)
	http.HandleFunc("/status", status)
	http.HandleFunc("/list", list)
	http.HandleFunc("/newFolder", newFolder)
	fmt.Println("Name server is running.")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
