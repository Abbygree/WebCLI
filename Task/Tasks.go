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

func JsonTaskOutput(writeGr []Task) {
	btResult, fileWriteErr := json.MarshalIndent(&writeGr, "", "  ")
	if fileWriteErr != nil {
		log.Fatal("Cannot encode to JSON", fileWriteErr)
	}
	jsonEncodeErr := ioutil.WriteFile("Tasks.json", btResult, 0777)
	if jsonEncodeErr != nil {
		log.Fatal("Cannot write data to file", jsonEncodeErr)
	}
}

func JsonTaskInput() (readTask []Task) {
	jsonTask, fileReadErr := ioutil.ReadFile("Tasks.json")
	if fileReadErr != nil {
		log.Fatal("Cannot read data from file", fileReadErr)
	}
	jsonDecodeErr := json.Unmarshal(jsonTask, &readTask)
	if jsonDecodeErr != nil {
		log.Fatal("Cannot decode from JSON", jsonDecodeErr)
	}
	return readTask
}
