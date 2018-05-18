package main

import (
	"github.com/google/uuid"
	"strings"
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
	chunks    []Chunk
}

var root *MyFile

func newChunk(index int) Chunk {
	tmp := Chunk{}
	tmp.id = uuid.New()
	for i, j := 0, 0; i < 4; i++ {
		if i != index {
			tmp.server[j] = i;
		}
		j++
	}
	return tmp
}

func newFile(path string, name string, isFile bool, chunkNum int) *MyFile {
	tmp := new(MyFile)

	if path == "" && name == "" {
		tmp.files = make([]*MyFile, 1)
		tmp.files[0] = tmp
		return tmp
	}

	tmp.id = uuid.New()
	tmp.isFile = isFile
	tmp.path = path
	tmp.basename = name

	//add to path
	currentPath := getFileFromPath(path)
	currentPath.files = append(currentPath.files, tmp)

	if isFile {
		tmp.chunks = make([]Chunk, chunkNum)

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

	return tmp
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

/*func main() {
	root = newFile("", "", false, 0)

	a:=newFile("", "a", false, 0)
	b:=newFile("", "b", true, 4)
	g:=getFileFromPath("a")
	c:= newFile("a", "c", false, 0)
	d:=newFile("a/c", "d", false, 0)
	e:=newFile("a/c", "e.233", true, 44)
	fmt.Println(a.id,b.id,c.id,d.id,e.id)
	f:=getFileFromPath("a/c/e.233")

	fmt.Println(f.id,g.id)
}*/
