package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"strings"
	"time"
)

type stupidData struct {
	SomeData     string
	SomeMoreData int
}

type dummyStruct struct {
	Data any
}

type KVindex struct {
	Key    string
	Offset int64
}

type LogEntry struct {
	PrefixMeta  string
	SuffixMeta  string
	Data        any
	EncodedData string
}

type Metadata struct {
	CreatedAt int64 // 8 bytes
	DataSize  int64
	Offset    int64
	Checksum  uint32 // 4 byte string
	Padding   string // filler
}

func (m *Metadata) Encode() string {
	buf := make([]byte, 1024)
	binary.LittleEndian.PutUint64(buf[0:8], uint64(m.CreatedAt))
	binary.LittleEndian.PutUint64(buf[8:16], uint64(m.DataSize))
	binary.LittleEndian.PutUint64(buf[16:24], uint64(m.Offset))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(m.Checksum))
	// 8bytes each for above stuff + 4bytes checksum for data, so 8*3 + 4 = 28

	copy(buf[28:], []byte(m.Padding))
	for i := 28 + len(m.Padding); i < 1024; i++ {
		buf[i] = 'X'
	}

	return string(buf)
}

func DecodeMetadata(data []byte) Metadata {
	if len(data) != 1024 {
		panic(fmt.Sprintf("Invalid data size: expected 1024 bytes, got %d bytes", len(data)))
	}
	return Metadata{
		CreatedAt: int64(binary.LittleEndian.Uint64(data[0:8])),
		DataSize:  int64(binary.LittleEndian.Uint64(data[8:16])),
		Offset:    int64(binary.LittleEndian.Uint64(data[16:24])),
		Checksum:  uint32(binary.LittleEndian.Uint32(data[24:28])),
		Padding:   string(data[28:]),
	}
}

// make the data into gob string
// from that gob string get the metadata
// /s + metadata + gob + /e

func (l *LogEntry) init(fileSize int64) {
	var gobBuf bytes.Buffer
	enc := gob.NewEncoder(&gobBuf)
	encapsulatedData := dummyStruct{
		Data: l.Data,
	}
	if err := enc.Encode(encapsulatedData); err != nil {
		panic(err)
	}

	l.EncodedData = gobBuf.String()
	checksum := getDataChecksum(l.EncodedData)

	prefixMetadata := Metadata{
		CreatedAt: time.Now().Unix(),
		DataSize:  int64(len(l.EncodedData)),
		Offset:    fileSize, // file ka size ? this is not threadsafe just so you know
		Checksum:  checksum,
		Padding:   "",
	}

	suffixMetadata := Metadata{
		CreatedAt: time.Now().Unix(),
		DataSize:  int64(len(l.EncodedData)),
		Offset:    2052 + int64(len(l.EncodedData)), // file ka size ? this is not threadsafe just so you know
		Padding:   "",
	}

	// in case of suffix meta, offset is the only valid thing, and it just means how much we have to jump back from \e to get to \s
	l.PrefixMeta = prefixMetadata.Encode()
	l.SuffixMeta = suffixMetadata.Encode()
}

func (l *LogEntry) getResult() string {
	return "/s" + l.PrefixMeta + l.EncodedData + l.SuffixMeta + "/e"
}

func getDataChecksum(data string) uint32 {
	return crc32.ChecksumIEEE([]byte(data))
}

func decodeEntry(encoded string) any {
	// log.Print("tryna decode: ", encoded)
	if !strings.HasPrefix(encoded, "/s") || !strings.HasSuffix(encoded, "/e") {
		panic("Invalid encoded format")
	}

	// Extract metadata (first 1024 bytes after /s)
	metaBytes := []byte(encoded[2:1026])
	metadata := DecodeMetadata(metaBytes)
	// log.Printf("Decoded Metadata: %+v\n", metadata)

	// Extract actual data using the size from metadata
	dataSize := metadata.DataSize
	dataBytes := []byte(encoded[1026 : 1026+dataSize])

	// get the checksum of dataBytes, compare it to metadata, if it does not matches, it means data is corrupted
	currChecksum := getDataChecksum(string(dataBytes))

	if currChecksum != metadata.Checksum {
		log.Print("xxxxxxx-we fucked up sire-xxxxxxx")
		return nil
	} else {
		log.Print("---file integrity looks good---")
	}
	// Decode gob data

	var decodedData dummyStruct
	dec := gob.NewDecoder(bytes.NewBuffer(dataBytes))
	if err := dec.Decode(&decodedData); err != nil {
		panic(err)
	}

	// log.Printf("Decoded Data: %+v\n", decodedData)
	return decodedData.Data
}

func AppendToFile(filePath string, data string) {
	// this appends for real LMAO
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	if _, err := file.WriteString(data); err != nil {
		panic(err)
	}
}

func AppendToFileSafe(filePath string, data string, WALPath string) {
	AppendToFile(WALPath, data)
	AppendToFile(filePath, data)

	// Verify the write by reading the last entry
	stat, err := os.Stat(filePath)
	if err != nil {
		panic(err)
	}

	readLen := len(data)
	lastEntry := ReadWithOffset(filePath, stat.Size()-int64(readLen), readLen)

	if lastEntry != data {
		// Revert changes if verification fails
		if err := os.Truncate(filePath, stat.Size()-int64(len(data))); err != nil {
			panic(err)
		}
		panic("Data verification failed")
	} else {
		// log.Print("----Data verified, all good-----")
	}

	AppendToFile(WALPath, "conf") // TODO: do something better
}

func ReadWithOffset(filePath string, offset int64, length int) string {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Seek to the given offset
	_, err = file.Seek(offset, 0) // io.SeekStart is 0
	if err != nil {
		panic(err)
	}

	// Read the given range
	buf := make([]byte, length)
	n, err := file.Read(buf)
	if err != nil {
		panic(err)
	}

	// log.Printf("Read %d bytes from offset %d: %v\n", n, offset, buf[:n])
	return string(buf[:n])
}

func ReadFileSequenatially(filePath string) {
	curOffset := 0

	for {
		// read in a 2kb buffer starting from curOffset
		encoded := ReadWithOffset(filePath, int64(curOffset), 2048)
		if len(encoded) < 2 {
			break
		}
		if !strings.HasPrefix(encoded, "/s") {
			panic("Invalid file format: entry should start with /s")
		}

		metaBytes := []byte(encoded[2:1026])
		metadata := DecodeMetadata(metaBytes)
		l, r := curOffset, int(metadata.DataSize)+2052
		entry := ReadWithOffset(filePath, int64(l), r)
		log.Print(decodeEntry(entry))
		curOffset = l + r
	}
}

func ReadFileSequenatiallyInReverse(filePath string) {
	stat, err := os.Stat(filePath)
	if err != nil {
		panic(err)
	}

	curOffset := stat.Size()
	for curOffset > 0 {
		// Read last 2KB or remaining file size
		readSize := int64(2048)
		if curOffset < readSize {
			readSize = curOffset
		}

		encoded := ReadWithOffset(filePath, curOffset-readSize, int(readSize))
		// log.Print(1)
		if !strings.HasSuffix(encoded, "/e") {
			panic("Invalid file format: entry should end with /e")
		}
		// log.Print(2)

		// Get the suffix metadata (1KB before /e)
		suffixMeta := DecodeMetadata([]byte(encoded[len(encoded)-1026 : len(encoded)-2]))
		// log.Print(3)
		// Calculate the start of current entry using offset from suffix metadata
		entryStart := curOffset - int64(suffixMeta.Offset)

		// Read the full entry
		log.Print("trying to seek to ", entryStart)
		entry := ReadWithOffset(filePath, entryStart, int(entryStart)+int(suffixMeta.Offset))
		// log.Print("lol")
		if !strings.HasPrefix(entry, "/s") {
			panic("Invalid file format: entry should start with /s")
		}

		decodeEntry(entry)
		curOffset = entryStart
	}
}

func NewLogEntry(data any, filePath string) *LogEntry {
	log := &LogEntry{
		Data: data,
	}
	stat, err := os.Stat(filePath)
	if err != nil {
		panic(err)
	}

	// if stat.Size() == 0 {
	// 	panic("wtf man")
	// }
	log.init(stat.Size())
	return log
}

// returns the decoded entry based off offset
func getEntryFromHead(filePath string, offset int64) any {
	// log.Print("hi ", filePath)
	// get the size of the file at filePath in bytes, print it along with offset
	// stat, err := os.Stat(filePath)
	// if err != nil {
	// 	log.Fatal("error agya bruh: ", err)
	// } else {
	// 	log.Print("---- ", stat.Size(), " -- ", offset, " done ")
	// }
	encoded := ReadWithOffset(filePath, offset, 2048)
	// log.Print("encoded shit is: ", encoded)
	if !strings.HasPrefix(encoded, "/s") {
		panic("Invalid file format: entry should start with /s")
	}

	metaBytes := []byte(encoded[2:1026])
	metadata := DecodeMetadata(metaBytes)
	// log.Print("metadata is: ", metadata)
	entry := ReadWithOffset(filePath, offset, int(metadata.DataSize)+2052)
	// log.Print("decoded entry is: ", decodeEntry(entry))
	// log.Print("tadaaaaaaaaa")
	// decodeEntry(entry)

	dataBytes := []byte(entry[1026 : 1026+metadata.DataSize])

	var decodedData dummyStruct
	dec := gob.NewDecoder(bytes.NewBuffer(dataBytes))
	if err := dec.Decode(&decodedData); err != nil {
		panic(err)
	}

	return decodedData.Data
}

// this needs to be run for all files!
func fileSanityChore(filePath string, WALPath string) {
	// Check if WAL file exists
	stat, err := os.Stat(WALPath)
	if err != nil {
		if os.IsNotExist(err) {
			return // No WAL file, nothing to do
		}
		panic(err)
	}

	// If file is empty or too small, clean it
	if stat.Size() < 4 {
		os.Truncate(WALPath, 0)
		return
	}

	// Read last 4 bytes to check for "conf"
	lastBytes := ReadWithOffset(WALPath, stat.Size()-4, 4)
	if lastBytes == "conf" {
		// Everything is confirmed, clean WAL
		os.Truncate(WALPath, 0)
		return
	} else if lastBytes[2:] != "/e" {
		// corrupted last WAL entry, clean WAL
		os.Truncate(WALPath, 0)
		return
	} else {
		// TODO: idk what this shit is, don't trust it
		// Get the last incomplete entry from WAL
		stat, err := os.Stat(WALPath)
		if err != nil {
			panic(err)
		}

		// Read last 2KB chunk to get metadata
		encoded := ReadWithOffset(WALPath, stat.Size()-2048, 2048)
		if !strings.HasSuffix(encoded, "/e") {
			panic("Invalid WAL format")
		}

		// Extract suffix metadata
		suffixMeta := DecodeMetadata([]byte(encoded[len(encoded)-1026 : len(encoded)-2]))

		// Clean up the main file from the offset
		mainFileStat, err := os.Stat(filePath)
		if err != nil {
			panic(err)
		}
		log.Print("wtf is this man eww: ", mainFileStat)

		// Truncate main file to remove incomplete write
		if err := os.Truncate(filePath, suffixMeta.Offset); err != nil {
			panic(err)
		}

		// Clean WAL
		os.Truncate(WALPath, 0)

		// TODO: retry the darn transaction and try to get conf in WAL
	}

	//
}

func ReadFileSequenatiallyAndReturnData(filePath string, startingOffset int) []any {
	curOffset := startingOffset
	dataColl := make([]any, 0)
	for {
		if int64(curOffset) == GetFileSize(filePath) {
			break
		}
		// read in a 2kb buffer starting from curOffset
		encoded := ReadWithOffset(filePath, int64(curOffset), 2048)
		if len(encoded) < 2 {
			break
		}
		if !strings.HasPrefix(encoded, "/s") {
			panic("Invalid file format: entry should start with /s")
		}

		metaBytes := []byte(encoded[2:1026])
		metadata := DecodeMetadata(metaBytes)
		l, r := curOffset, int(metadata.DataSize)+2052
		entry := ReadWithOffset(filePath, int64(l), r)
		// log.Print("entry is: ", entry)
		dataColl = append(dataColl, decodeEntry(entry))
		curOffset = l + r

	}

	return dataColl
}

func GetFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// func main() {
// 	filePath := "/Users/nubskr/pxrs/persisto/log.log"
// 	WALPath := "/Users/nubskr/pxrs/persisto/WAL.log"
// 	startupChore(filePath, WALPath)

// data := stupidData{
// 	SomeData:     "hie daddy UwU",
// 	SomeMoreData: 282829,
// }

// 	// log := NewLogEntry(data, filePath)

// 	// encoded_entry := log.getResult()

// 	// AppendToFileSafe(filePath, encoded_entry, WALPath)

// 	// log.Print("=======start")
// 	// ReadFileSequenatiallyInReverse(filePath)
// }

/*
\s(1kb metadata)(actual data)(1 kb metadata)\e

note that the `padding` is only relevant for WAL, it's NOT relevant in actual data, please remove it from there in future to avoid any confusions
*/

//TODO: make sure to have some sort of init function which rebuilds all KV indexes at startup
