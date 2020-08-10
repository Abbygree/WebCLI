package main

import (
	"WebCLI/Group"
	"WebCLI/Task"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	sort2 "sort"
	"strconv"
)

var Groups []Group.Group
var Tasks []Task.Task

func main() {
	http.HandleFunc("/groups", GetGroups)
	http.HandleFunc("/groups", GetGroups)

}

func JsonGroupInput() []byte {
	jsonGr, err := ioutil.ReadFile("Groups.json")
	if err != nil {
		log.Fatal("Cannot read data from file", err)
	}
	err = json.Unmarshal(jsonGr, &Groups)
	if err != nil {
		log.Fatal("Cannot decode from JSON", err)
	}
	return jsonGr
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

func JsonTaskInput() []byte {
	jsonGr, err := ioutil.ReadFile("Tasks.json")
	if err != nil {
		log.Fatal("Cannot read data from file", err)
	}
	err = json.Unmarshal(jsonGr, &Groups)
	if err != nil {
		log.Fatal("Cannot decode from JSON", err)
	}
	return jsonGr
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
	sort := req.URL.Query().Get("sort")
	limit, err := strconv.Atoi(req.URL.Query().Get("limit"))
	if err != nil{
		limit = 0
	}
	if sort == ""{
		jsonGr := JsonGroupInput()
		var unJsonedGr []Group.Group
		_ = json.Unmarshal(JsonTaskInput(), &unJsonedGr)
		if err != nil{
			limit = len(unJsonedGr)
		}
		unJsonedGr = unJsonedGr[:limit]
		fmt.Fprint(w, jsonGr)
	}else {
		GetGroupsSort(&w, req, sort, limit)
	}

}

func GetGroupsSort(w *http.ResponseWriter, req *http.Request, sort string, limit int) {

	//unmarshall json file to groups' slice and ascending sort by name
	var unJsonedGr, parentsGr, childsGr, childGr, grandChildsGr []Group.Group
	_ = json.Unmarshal(JsonTaskInput(), &unJsonedGr)
	if limit == 0{
		limit = len(unJsonedGr)
	}
	sort2.SliceStable(&unJsonedGr, func(i, j int) bool { return unJsonedGr[i].GroupName < unJsonedGr[j].GroupName })

	//create the parents and childs subslices
	for i := 0; i < len(unJsonedGr); i++ {
		if unJsonedGr[i].GroupID == 0 {
			parentsGr = append(parentsGr, unJsonedGr[i])
		} else {
			childGr = append(childGr, unJsonedGr[i])
		}
	}

	//create the childs and grandchilds subslices
	for i := 0; i < len(childGr); i++{
		for j := 0; j < len(parentsGr); j++{
			if (childGr[i].ParentID == parentsGr[j].GroupID) && (parentsGr[j].ParentID != 0){
				grandChildsGr = append(grandChildsGr, childGr[i])
			}else{
				childsGr = append(childsGr, childGr[i])
			}
		}
	}

	switch sort {
	case "name":
		subUnJsonedGr := unJsonedGr[:limit]
		jsonGr, _ := json.MarshalIndent(&subUnJsonedGr, "", "  ")
		fmt.Fprint(*w, string(jsonGr))
		break
	case "parents_first":
		unJsonedGr = append(parentsGr, childsGr...)
		unJsonedGr = append(unJsonedGr, grandChildsGr...)
		subUnJsonedGr := unJsonedGr[:limit]
		jsonGr, _ := json.MarshalIndent(&subUnJsonedGr, "", "  ")
		fmt.Fprint(*w, string(jsonGr))
		break
	case "parent_with_childs":
		var pwcGrJson []Group.Group

		for i := 0; i < len(parentsGr); i++ {
			pwcGrJson = append(pwcGrJson, parentsGr[i])

			for j := 0; j < len(childGr); j++ {
				if childsGr[j].ParentID == parentsGr[i].GroupID {
					pwcGrJson = append(pwcGrJson, childsGr[j])

					for k := 0; k < len(grandChildsGr); k++{
						if grandChildsGr[k].ParentID == childsGr[j].GroupID{
							pwcGrJson = append(pwcGrJson, grandChildsGr[k])
						}
					}
				}
			}
		}
		jsonGr, _ := json.MarshalIndent(&pwcGrJson, "", "  ")
		fmt.Fprint(*w, string(jsonGr))
		break
	default:
		(*w).WriteHeader(http.StatusBadRequest)
		break
	}

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

func PostNewTasks(w http.ResponseWriter, req *http.Request) {

}

func PutTasksByID(w http.ResponseWriter, req *http.Request) {

}

func GetTasksGroupByID(w http.ResponseWriter, req *http.Request) {
	//type := req.URL.Query().Get("type")
}

func PostTasksByID(w http.ResponseWriter, req *http.Request) {

}

func GetStatToday(w http.ResponseWriter, req *http.Request) {

}

func GetStatYesterday(w http.ResponseWriter, req *http.Request) {

}

func GetStatWeek(w http.ResponseWriter, req *http.Request) {

}

func GetStatMonth(w http.ResponseWriter, req *http.Request) {

}
