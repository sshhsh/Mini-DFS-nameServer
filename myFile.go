package main

import (
	"github.com/google/uuid"
	"strings"
	"errors"
	"fmt"
)

type Chunk struct {
	id     uuid.UUID
	server [3]int
}

type MyFile struct {
	id        uuid.UUID
	isFile    bool
	path      string
	basename  string
	extension string
	filename  string
	files     []*MyFile
	chunks    []*Chunk
}

var root *MyFile

func newChunk(index int) *Chunk {
	tmp := new(Chunk)
	tmp.id = uuid.New()
	for i, j := 0, 0; i < 4 && j < 3; i++ {
		if i != index {
			tmp.server[j] = i
			j++
		}
	}
	return tmp
}

func exists(path string, name string) bool {
	currentPath := getFileFromPath(path)
	for _, previousFile := range currentPath.files {
		if name == previousFile.basename {
			return true
		}
	}
	return false
}

func newFile(path string, name string, isFile bool, chunkNum int) (*MyFile, error) {
	tmp := new(MyFile)

	//for creating root
	if path == "" && name == "" {
		tmp.files = make([]*MyFile, 1)
		tmp.files[0] = tmp
		return tmp, nil
	}

	tmp.id = uuid.New()
	tmp.isFile = isFile
	tmp.path = path
	tmp.basename = name

	//add to path
	currentPath := getFileFromPath(path)

	//check for same file
	for _, previousFile := range currentPath.files {
		if name == previousFile.basename {
			return nil, errors.New("file existed")
		}
	}

	currentPath.files = append(currentPath.files, tmp)

	if isFile {
		//tmp.chunks = make([]Chunk, chunkNum)

		index := strings.LastIndex(name, ".")
		if index != -1 {
			tmp.filename = name[0:index]
			tmp.extension = name[index+1:]
		} else {
			tmp.filename = name
			tmp.extension = ""
		}
	} else {
		tmp.files = make([]*MyFile, 1)
		tmp.files[0] = tmp
	}

	return tmp, nil
}

func getFileFromPath(path string) *MyFile {
	if path == "" {
		return root
	}

	paths := strings.Split(path, "/")
	currentPath := root
	flag := false
	for i, dir := range paths {
		for _, file := range currentPath.files {
			if file.basename == dir {
				currentPath = file
				if i == len(paths)-1 {
					flag = true
				}
				break
			}
		}
	}
	if flag {
		return currentPath
	}
	return nil
}

func main() {
	root, _ = newFile("", "", false, 0)

	a, _ := newFile("", "a", false, 0)
	_, err := newFile("", "a", true, 4)

	g:=getFileFromPath("a")
	c, _ := newFile("a", "c", false, 0)
	d, _ := newFile("a/c", "d", false, 0)
	e, _ := newFile("a/c", "e.233", true, 44)
	fmt.Println(a.id, err, c.id, d.id, e.id)
	f:=getFileFromPath("a/c/e.233")
	h := exists("a/c", "e.233")

	fmt.Println(f.id, g.id, h)
}
