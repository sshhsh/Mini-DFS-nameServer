package main

import (
	"net/http"
	"log"
	"strings"
	"fmt"
)

var dataServer [4]string
var dataServerStatus [4]int

const RUNNING int = 1
const NONE int = 0
const ERROR int = 4
const RECOVERING int = 7

func upload(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	//TODO
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
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/download", download)
	http.HandleFunc("/register", register)
	http.HandleFunc("/status", status)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
