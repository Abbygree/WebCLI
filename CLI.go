package main

import (
	"WebCLI/Group"
	"WebCLI/Task"
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	sort2 "sort"
	"strconv"
)

var Groups []Group.Group
var Tasks []Task.Task

var jsonGroups []byte

func main() {
	jsonGroups = JsonGroupInput()
	router := mux.NewRouter()
	router.HandleFunc("/groups", GetGroups).Methods(http.MethodGet)

	http.ListenAndServe(":8181", router)

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
	req.ParseForm()
	sort, srtOk := req.Form["sort"]
	limitstr, limOk := req.Form["limit"]
	var limit int
	if !limOk {
		limit = 0
	} else {
		limit, _ = strconv.Atoi(limitstr[0])
		_ = limit
	}

	if !srtOk {
		var unJsonedGr []Group.Group
		_ = json.Unmarshal(jsonGroups, &unJsonedGr)
		if !limOk {
			limit = len(unJsonedGr)
		}
		unJsonedGr = unJsonedGr[:limit]
		json.NewEncoder(w).Encode(unJsonedGr)
	} else {
		GetGroupsSort(&w, req, sort[0], limit)
	}

}

func GetGroupsSort(w *http.ResponseWriter, req *http.Request, sort string, limit int) {

	//unmarshall json file to groups' slice and ascending sort by name
	var unJsonedGr, parentsGr, childsGr, childGr, grandChildsGr []Group.Group
	_ = json.Unmarshal(jsonGroups, &unJsonedGr)
	if limit == 0 {
		limit = len(unJsonedGr)
	}
	sort2.SliceStable(unJsonedGr, func(i, j int) bool { return unJsonedGr[i].GroupName < unJsonedGr[j].GroupName })

	//create the parents and childs subslices
	for i := 0; i < len(unJsonedGr); i++ {
		if unJsonedGr[i].ParentID == 0 {
			parentsGr = append(parentsGr, unJsonedGr[i])
		} else {
			childGr = append(childGr, unJsonedGr[i])
		}
	}

	//create the childs and grandchilds subslices
	for i := 0; i < len(childGr); i++ {
		if grContain(parentsGr, childGr[i]) {
			childsGr = append(childsGr, childGr[i])
		} else {
			grandChildsGr = append(grandChildsGr, childGr[i])
		}
	}

	switch sort {
	case "name":
		subUnJsonedGr := unJsonedGr[:limit]
		json.NewEncoder(*w).Encode(subUnJsonedGr)
		break
	case "parents_first":
		unJsonedGr = append(parentsGr, childsGr...)
		unJsonedGr = append(unJsonedGr, grandChildsGr...)
		subUnJsonedGr := unJsonedGr[:limit]
		json.NewEncoder(*w).Encode(subUnJsonedGr)
		break
	case "parent_with_childs":
		var pwcGrJson []Group.Group

		for i := 0; i < len(parentsGr); i++ {
			pwcGrJson = append(pwcGrJson, parentsGr[i])

			for j := 0; j < len(childsGr); j++ {
				if childsGr[j].ParentID == parentsGr[i].GroupID {
					pwcGrJson = append(pwcGrJson, childsGr[j])

					for k := 0; k < len(grandChildsGr); k++ {
						if grandChildsGr[k].ParentID == childsGr[j].GroupID {
							pwcGrJson = append(pwcGrJson, grandChildsGr[k])
						}
					}
				}
			}
		}
		json.NewEncoder(*w).Encode(pwcGrJson)
		break
	default:
		(*w).WriteHeader(http.StatusBadRequest)
		break
	}
}

func grContain(arrGr []Group.Group, contGr Group.Group) (result bool) {
	for i := 0; i < len(arrGr); i++ {
		result = result || arrGr[i].GroupID == contGr.ParentID
	}
	return result
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
