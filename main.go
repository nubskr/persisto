package main

import (
	"encoding/gob"
	"log"
	"os"
	"path/filepath"
)

var (
	keyIndex   = make(map[string]int64) // In-memory key-offset index for KV store
	queueIndex = make(map[string]int64) // In-memory queue read offsets
	baseDir    = ".persisto"            // base dir
)

func getFilePaths(varName string) (string, string) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		panic(err)
	}
	mainFile := filepath.Join(baseDir, varName+"_main")
	walFile := filepath.Join(baseDir, varName+"_WAL")
	for _, file := range []string{mainFile, walFile} {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			if _, err := os.Create(file); err != nil {
				panic(err)
			}
		}
	}
	return mainFile, walFile
}

func PushToQueue(queueName string, data any) error {
	filePath, WALPath := getFilePaths(queueName + "_queue")

	// Append data to main log
	// offset, _ := os.Stat(filePath) // Get current file size as new offset
	AppendToFileSafe(filePath, NewLogEntry(data, filePath).getResult(), WALPath)

	// Store queue offset
	// queueOffsetFile, _ := getFilePaths(queueName + "_offset")
	// AppendToFileSafe(queueOffsetFile, NewLogEntry(offset.Size(), queueOffsetFile).getResult(), WALPath)

	// Update in-memory queue index
	// queueIndex[queueName] = offset.Size()

	return nil
}

func PopQueue(queueName string) any {
	filePath, _ := getFilePaths(queueName + "_queue")

	offset, exists := queueIndex[queueName]
	if !exists {
		return nil
		// queueIndex[queueName] = 0
	}
	// log.Print("offset is: ", queueIndex[queueName])
	fileInfo, _ := os.Stat(filePath)
	if offset >= fileInfo.Size() {
		return nil
	}

	metaBytes := ReadWithOffset(filePath, offset+2, 1024)
	metadata := DecodeMetadata([]byte(metaBytes))

	data := getEntryFromHead(filePath, offset)

	// move forward
	queueIndex[queueName] = offset + 2052 + metadata.DataSize
	SetMapVal(queueName+"_offset", queueIndex[queueName])

	return data
}

func SetMapVal(key string, data any) error {
	filePath, WALPath := getFilePaths(key + "_KV")
	offset, _ := os.Stat(filePath)
	AppendToFileSafe(filePath, NewLogEntry(data, filePath).getResult(), WALPath)
	indexFile, _ := getFilePaths(key + "_index")
	AppendToFileSafe(indexFile, NewLogEntry(offset.Size(), indexFile).getResult(), WALPath)
	keyIndex[key] = offset.Size()
	return nil
}

func GetMapVal(key string) any {
	filePath, _ := getFilePaths(key)
	log.Print("filepath is: ", filePath)
	offset, exists := keyIndex[key]
	if !exists {
		return nil // Key not there
	}
	return getEntryFromHead(filePath, offset)
}

func main() {
	gob.Register(stupidData{})
	// err := SetMapVal("key1", "val1")
	lolol := stupidData{
		SomeData:     "hie daddy UwU",
		SomeMoreData: 2828299,
	}
	// err = SetMapVal("key2", lolol)
	// if err != nil {
	// 	panic(err)
	// }
	// log.Print("====")
	// log.Print(GetMapVal("key1"))
	// log.Print(GetMapVal("key2"))
	err := PushToQueue("Q1", lolol)
	err = PushToQueue("Q1", "what am I doing with my life")
	if err != nil {
		panic(err)
	}

	log.Print(PopQueue("Q1"))
	log.Print(PopQueue("Q1"))

}
