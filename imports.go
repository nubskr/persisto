package main

import (
	"log"
	"path/filepath"
)

// send all at once ? can they even handle it ? yeah, they can, just send the data, not the whole log entry

func ImportQueueData(queueName string) []any {
	queueMain := filepath.Join(baseDir, queueName+"_queue"+"_main")
	queueWAL := filepath.Join(baseDir, queueName+"_queue"+"_WAL")

	fileSanityChore(queueMain, queueWAL)

	// just load up the queue offset in memory from the queue name
	offsetVal := GetMapVal("_" + queueName + "_offset")
	if offsetVal == nil {
		log.Print("queue does not exists")
		return nil // queue does not exists
	}
	log.Print(offsetVal, GetFileSize(queueMain))
	res := ReadFileSequenatiallyAndReturnData(queueMain, int(offsetVal.(int64)))
	return res
	// log.Print(res)
	// resCont := make([]any, 0)
	// for _, item := range res {
	// 	resCont = append(resCont, item)
	// }
	// return resCont
}

func ImportKVData() map[string]any {
	kvMain := filepath.Join(baseDir, "KV_main")

	// keyIndex = make(map[string]int64)
	resMap := make(map[string]any)

	for k := range keyIndex {
		resMap[k] = getEntryFromHead(kvMain, keyIndex[k])
	}

	return resMap
}
