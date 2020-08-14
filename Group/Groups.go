package Group

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type Group struct {
	GroupName        string `json:"group_name"`
	GroupDescription string `json:"group_description"`
	GroupID          int    `json:"group_id"`
	ParentID         int    `json:"parent_id"`
}

func JsonGroupInput() (readGr []Group) {
	jsonGr, fileReadErr := ioutil.ReadFile("Groups.json")
	if fileReadErr != nil {
		log.Fatal("Cannot read data from file", fileReadErr)
	}
	jsonDecodeErr := json.Unmarshal(jsonGr, &readGr)
	if jsonDecodeErr != nil {
		log.Fatal("Cannot decode from JSON", jsonDecodeErr)
	}
	return readGr
}

func JsonGroupOutput(writeGr []Group) {
	btResult, fileWriteErr := json.MarshalIndent(&writeGr, "", "  ")
	if fileWriteErr != nil {
		log.Fatal("Cannot encode to JSON", fileWriteErr)
	}
	jsonEncodeErr := ioutil.WriteFile("Groups.json", btResult, 0777)
	if jsonEncodeErr != nil {
		log.Fatal("Cannot write data to file", jsonEncodeErr)
	}
}
