package Task

import (
	"WebCLI/Service"
	"encoding/json"
	"io/ioutil"
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
	var startTime, endTime time.Time
	var err error
	var msg string
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.NonRequestErrExecLog("JsonTaskOutput", endTime.Sub(startTime), err, msg)
	}()

	btResult, fileWriteErr := json.MarshalIndent(&writeGr, "", "  ")
	if fileWriteErr != nil {
		err = fileWriteErr
		msg = "Cannot encode to JSON"
	}
	jsonEncodeErr := ioutil.WriteFile("Task/Tasks.json", btResult, 0777)
	if jsonEncodeErr != nil {
		err = jsonEncodeErr
		msg = "Cannot write data to file"
	}
}

func JsonTaskInput() (readTask []Task) {
	var startTime, endTime time.Time
	var err error
	var msg string
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.NonRequestErrExecLog("JsonTaskInput", endTime.Sub(startTime), err, msg)
	}()

	jsonTask, fileReadErr := ioutil.ReadFile("Task/Tasks.json")
	if fileReadErr != nil {
		err = fileReadErr
		msg = "Cannot read data from file"
	}
	jsonDecodeErr := json.Unmarshal(jsonTask, &readTask)
	if jsonDecodeErr != nil {
		err = jsonDecodeErr
		msg = "Cannot decode from JSON"
	}
	return readTask
}
