package main

// this thing is responsible for serialization as well ?
// fuck me, please don't serialize and deserialize stuff man wtf

// channel of what ?

// need to verify type somehow maybe

// data needs to be serialized as it is being inserted in a text file

// NEEDS to be a single threaded thing

var START_TOKEN string = "/s"
var END_TOKEN string = "/e"

type Task struct {
	dataType string
	data     any
}

// append payload: START_TOKEN + (InnerPayload.end_point) + (InnerPayload.serialized_data) + END_TOKEN

// verify changes
type InnerPayload struct {
	end_point       int
	serialized_data string
}

var taskQueue chan Task = make(chan Task, 100)

func serialize(data any) {
	// data is a struct, serialize it in gob
}

func saveToFile(task Task) {
	switch task.dataType {
	case "queue":

	case "map":

	}
}

func PushToQueue(queue string, c chan any, data any) error {
	c <- data
	// serialize it and push to file
	return nil
}

func PopQueue(queue string, c chan any) any {
	data := <-c
	// serialize it and push to file
	return data
}

func SetMapVal(key string, value string) error {

}
