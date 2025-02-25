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

	tmpData := GetMapVal("_" + queueName + "_offset")
	if tmpData == nil {
		// first time queue init
	} else {
		queueIndex[queueName] = tmpData.(int64)
	}
	offset, exists := queueIndex[queueName]
	if !exists {
		queueIndex[queueName] = 0
		SetMapVal("_"+queueName+"_offset", queueIndex[queueName])
	}
	log.Print("queue offset is: ", offset)
	// Store queue offset
	// queueOffsetFile, _ := getFilePaths(queueName + "_offset")
	// AppendToFileSafe(queueOffsetFile, NewLogEntry(offset.Size(), queueOffsetFile).getResult(), WALPath)

	// Update in-memory queue index
	// queueIndex[queueName] = offset.Size()

	return nil
}

func PopQueue(queueName string) any {
	filePath, _ := getFilePaths(queueName + "_queue")

	queueIndex[queueName] = GetMapVal("_" + queueName + "_offset").(int64)
	log.Print("queueIndex is: ", queueIndex[queueName])
	offset, exists := queueIndex[queueName]
	if !exists {
		log.Print("does not exists babe")
		return nil
	}

	if offset >= GetFileSize(filePath) {
		return nil
	}

	metaBytes := ReadWithOffset(filePath, offset+2, 1024)
	metadata := DecodeMetadata([]byte(metaBytes))

	data := getEntryFromHead(filePath, offset)

	// move forward
	queueIndex[queueName] = offset + 2052 + metadata.DataSize
	log.Print("queueIndex updated to: ", queueIndex[queueName])
	SetMapVal("_"+queueName+"_offset", queueIndex[queueName])

	return data
}

func SetMapVal(key string, data any) error {
	filePath, WALPath := getFilePaths("KV")
	offset, _ := os.Stat(filePath)
	AppendToFileSafe(filePath, NewLogEntry(data, filePath).getResult(), WALPath)
	indexFile, _ := getFilePaths("indexes")
	AppendToFileSafe(indexFile, NewLogEntry(KVindex{Key: key, Offset: offset.Size()}, indexFile).getResult(), WALPath)
	keyIndex[key] = offset.Size()
	return nil
}

func GetMapVal(key string) any {
	// filePath, _ := getFilePaths(key)
	filePath := "./.persisto/KV_main"
	// log.Print("filepath is: ", filePath)
	offset, exists := keyIndex[key]
	if !exists {
		return nil // Key not there
	}
	return getEntryFromHead(filePath, offset)
}

func startupChore() {
	// TODO: check if those files even exist or not
	indexesMain := filepath.Join(baseDir, "indexes_main")
	indexesWAL := filepath.Join(baseDir, "indexes_WAL")
	kvMain := filepath.Join(baseDir, "KV_main")
	kvWAL := filepath.Join(baseDir, "KV_WAL")

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		panic(err)
	}

	for _, file := range []string{indexesMain, indexesWAL, kvMain, kvWAL} {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			if _, err := os.Create(file); err != nil {
				panic(err)
			}
		}
	}

	fileSanityChore(indexesMain, indexesWAL)
	fileSanityChore(kvMain, kvWAL)

	res := ReadFileSequenatiallyAndReturnData(indexesMain, 0)
	log.Print(res)

	keyIndex = make(map[string]int64)

	// TODO: not effecient, read from the bottom
	for _, item := range res {
		actualItem := item.(KVindex)
		keyIndex[actualItem.Key] = actualItem.Offset
	}

	log.Print("----restore done----")
}

func main() {
	gob.Register(stupidData{})
	gob.Register(KVindex{})

	startupChore()

	// err := SetMapVal("key1", "val3")
	// lolol := stupidData{
	// 	SomeData:     "hie daddy UwU",
	// 	SomeMoreData: 2828299,
	// }
	// err = SetMapVal("key2", lolol)
	// if err != nil {
	// 	panic(err)
	// }
	// log.Print("====")
	// // log.Print(GetMapVal("key1"))
	// // log.Print(GetMapVal("key2"))
	// err = PushToQueue("Q1", "1")
	// err = PushToQueue("Q1", "2")
	// err = PushToQueue("Q1", "3")
	// if err != nil {
	// 	panic(err)
	// }

	// log.Print(PopQueue("Q1"))
	// log.Print(PopQueue("Q1"))
	// log.Print("------")
	// ImportKVData()
	// log.Print(ImportQueueData("Q1"))
	// log.Print(ImportKVData())

}
