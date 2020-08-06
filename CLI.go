package main

import (
	"WebCLI/Group"
	"WebCLI/Task"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var Groups []Group.Group
var Tasks  []Task.Task

func main() {

}

func JsonGroupInput() {
	jsonGr, err := ioutil.ReadFile("Groups.json")
	if err != nil {
		log.Fatal("Cannot read data from file", err)
	}
	err = json.Unmarshal(jsonGr, &Groups)
	if err != nil {
		log.Fatal("Cannot decode from JSON", err)
	}
}

func JsonGroupOutput() {
	btResult, err := json.MarshalIndent(&Groups, "", "  ")
	if err != nil {
		log.Fatal("Cannot encode to JSON", err)
	}
	err = ioutil.WriteFile("Groups.json", btResult, 0777)
	if err != nil {
		log.Fatal("Cannot write data to file", err)
	}
}

func JsonTaskInput() {
	jsonGr, err := ioutil.ReadFile("Tasks.json")
	if err != nil {
		log.Fatal("Cannot read data from file", err)
	}
	err = json.Unmarshal(jsonGr, &Groups)
	if err != nil {
		log.Fatal("Cannot decode from JSON", err)
	}
}

func JsonTaskOutput() {
	btResult, err := json.MarshalIndent(&Tasks, "", "  ")
	if err != nil {
		log.Fatal("Cannot encode to JSON", err)
	}
	err = ioutil.WriteFile("Tasks.json", btResult, 0777)
	if err != nil {
		log.Fatal("Cannot write data to file", err)
	}
}

func GetGroups(w http.ResponseWriter, req *http.Request) {

}

func GetGroupsSort(w http.ResponseWriter, req *http.Request) {
	//sort := req.URL.Query().Get("sort")
	//limit :=req.URL.Query().Get("limit")

}

func GetGroupTopParents(w http.ResponseWriter, req *http.Request) {

}

func GetGroupByID(w http.ResponseWriter, req *http.Request) {

}

func GetGroupChildsByID(w http.ResponseWriter, req *http.Request) {

}

func PostNewGroup(w http.ResponseWriter, req *http.Request) {

}

func PutGroupByID(w http.ResponseWriter, req *http.Request) {

}

func DeleteGroupByID(w http.ResponseWriter, req *http.Request) {

}

func GetTasksSort(w http.ResponseWriter, req *http.Request) {
	//sort := req.URL.Query().Get("sort")
	//limit :=req.URL.Query().Get("limit")
	//typeof := req.URL.Query().Get("type")

}
