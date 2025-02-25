package main

import (
	"path/filepath"
)

// send all at once ? can they even handle it ? yeah, they can, just send the data, not the whole log entry

func ImportQueueData(queueName string) []any {
	// check for corruptions and recover if needed
	queueMain := filepath.Join(baseDir, queueName+"_queue"+"_main")
	queueWAL := filepath.Join(baseDir, queueName+"_queue"+"_WAL")

	startupChore(queueMain, queueWAL)

	// just load up the queue offset in memory from the queue name
	offsetVal := GetMapVal(queueName + "_offset")
	res := ReadFileSequenatiallyAndReturnData(queueMain, offsetVal.(int))

	resCont := make([]any, 0)
	// log.Print(res)
	for _, item := range res {
		resCont = append(resCont, item)
		// actualItem := item.(KVindex)
		// resMap[actualItem.Key] = getEntryFromHead(kvMain, actualItem.Offset)
	}
	// just start reading from offset, that's it
	return resCont
}

func ImportKVData() map[string]any {
	indexesMain := filepath.Join(baseDir, "indexes_main")
	indexesWAL := filepath.Join(baseDir, "indexes_WAL")
	kvMain := filepath.Join(baseDir, "KV_main")
	kvWAL := filepath.Join(baseDir, "KV_WAL")

	startupChore(indexesMain, indexesWAL)
	startupChore(kvMain, kvWAL)

	res := ReadFileSequenatiallyAndReturnData(indexesMain, 0)

	// keyIndex = make(map[string]int64)
	resMap := make(map[string]any)

	// TODO: not effecient, read from the bottom
	for _, item := range res {
		actualItem := item.(KVindex)
		resMap[actualItem.Key] = getEntryFromHead(kvMain, actualItem.Offset)
	}

	return resMap
}
