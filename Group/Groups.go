package Group

import (
	"WebCLI/Service"
	"encoding/json"
	"io/ioutil"
	"time"
)

type Group struct {
	GroupName        string `json:"group_name"`
	GroupDescription string `json:"group_description"`
	GroupID          int    `json:"group_id"`
	ParentID         int    `json:"parent_id"`
}

func JsonGroupInput() (readGr []Group) {
	var startTime, endTime time.Time
	var err error
	var msg string
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.NonRequestErrExecLog("JsonGroupInput", endTime.Sub(startTime), err, msg)
	}()

	jsonGr, fileReadErr := ioutil.ReadFile("Group/Groups.json")
	if fileReadErr != nil {
		err = fileReadErr
		msg = "Cannot read data from file"
	}
	jsonDecodeErr := json.Unmarshal(jsonGr, &readGr)
	if jsonDecodeErr != nil {
		err = jsonDecodeErr
		msg = "Cannot decode from JSON"
	}
	return readGr
}

func JsonGroupOutput(writeGr []Group) {
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
	jsonEncodeErr := ioutil.WriteFile("Group/Groups.json", btResult, 0777)
	if jsonEncodeErr != nil {
		err = jsonEncodeErr
		msg = "Cannot write data to file"
	}
}
