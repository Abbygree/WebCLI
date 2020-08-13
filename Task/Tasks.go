package Task

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"time"
)

type Task struct {
	TaskID      string    `json:"task_id"`
	GroupID     int       `json:"group_id"`
	Task        string    `json:"task"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at"`
}

func JsonTaskOutput(writeGr []Task) /*(fileWriteErr error, jsonEncodeErr error)*/ {
	btResult, fileWriteErr := json.MarshalIndent(&writeGr, "", "  ")
	if fileWriteErr != nil {
		log.Fatal("Cannot encode to JSON", fileWriteErr)
	}
	jsonEncodeErr := ioutil.WriteFile("Task.json", btResult, 0777)
	if jsonEncodeErr != nil {
		log.Fatal("Cannot write data to file", jsonEncodeErr)
	}
	/*return fileWriteErr, jsonEncodeErr*/
}

func JsonTaskInput() ( /*fileReadErr error, jsonDecodeErr error, */ readTask []Task) {
	jsonGr, fileReadErr := ioutil.ReadFile("Tasks.json")
	if fileReadErr != nil {
		log.Fatal("Cannot read data from file", fileReadErr)
	}
	jsonDecodeErr := json.Unmarshal(jsonGr, &readTask)
	if jsonDecodeErr != nil {
		log.Fatal("Cannot decode from JSON", jsonDecodeErr)
	}
	return /*fileReadErr, jsonDecodeErr,*/ readTask
}
