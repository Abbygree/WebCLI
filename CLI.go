package main

import (
	"WebCLI/Group"
	"WebCLI/Task"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	sort2 "sort"
	"strconv"
	"time"
)

var Tasks []Task.Task
var Groups []Group.Group

func main() {
	Groups = Group.JsonGroupInput()
	Tasks = Task.JsonTaskInput()
	router := mux.NewRouter()
	router.HandleFunc("/groups", GetGroups).Methods(http.MethodGet)
	router.HandleFunc("/group/top_parents", GetGroupTopParents).Methods(http.MethodGet)
	router.HandleFunc("/group/{id}", GetGroupByID).Methods(http.MethodGet)
	router.HandleFunc("/group/childs/{id}", GetGroupChildsByID).Methods(http.MethodGet)
	router.HandleFunc("/group/new", PostNewGroup).Methods(http.MethodPost)
	router.HandleFunc("/group/{id}", PutGroupByID).Methods(http.MethodPut)
	router.HandleFunc("/group/{id}", DeleteGroupByID).Methods(http.MethodDelete)
	router.HandleFunc("/tasks", GetTasksSort).Methods(http.MethodGet)
	router.HandleFunc("/tasks/new", PostNewTasks).Methods(http.MethodPost)
	router.HandleFunc("/tasks/{id}", PutTasksByID).Methods(http.MethodPut)
	http.ListenAndServe(":8181", router)
	defer Group.JsonGroupOutput(Groups)
}

func taskNGrIDToHashToString5(task string, grID int) (str string) {
	task += strconv.Itoa(grID)
	hsh := md5.Sum([]byte(task))
	str = fmt.Sprintf("%x", hsh)
	return str[:6]
}

func GetGroups(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	sort, srtOk := req.Form["sort"]
	limitstr, limOk := req.Form["limit"]
	var limit int
	if !limOk {
		limit = len(Groups)
	} else {
		limit, _ = strconv.Atoi(limitstr[0])
		if (limit > len(Groups)) || (limit == 0) {
			limit = len(Groups)
		}
	}

	if !srtOk {
		unJsonedGr := Groups[:limit]
		w.Header().Set("content-type", "application/json")
		err := json.NewEncoder(w).Encode(unJsonedGr)
		if err != nil {
			log.Fatal("Cannot decode from JSON", err)
			return
		}
	} else {
		GetGroupsSort(&w, req, sort[0], limit)
	}

}

func GetGroupsSort(w *http.ResponseWriter, req *http.Request, sort string, limit int) {

	//unmarshall json file to groups' slice and ascending sort by name
	var unJsonedGr, parentsGr, childsGr, childGr, grandChildsGr []Group.Group
	unJsonedGr = Groups
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
		(*w).Header().Set("content-type", "application/json")
		err := json.NewEncoder(*w).Encode(subUnJsonedGr)
		if err != nil {
			log.Fatal("Cannot decode from JSON", err)
			return
		}
		break
	case "parents_first":
		unJsonedGr = append(parentsGr, childsGr...)
		unJsonedGr = append(unJsonedGr, grandChildsGr...)
		subUnJsonedGr := unJsonedGr[:limit]
		(*w).Header().Set("content-type", "application/json")
		err := json.NewEncoder(*w).Encode(subUnJsonedGr[:limit])
		if err != nil {
			log.Fatal("Cannot decode from JSON", err)
			return
		}
		break
	case "parent_with_childs":
		var pwcGrJson []Group.Group

		for i := 0; i < len(parentsGr); i++ {
			pwcGrJson = append(pwcGrJson, parentsGr[i])

			for j := 0; j < len(childsGr); j++ {
				//Search childs
				if childsGr[j].ParentID == parentsGr[i].GroupID {
					pwcGrJson = append(pwcGrJson, childsGr[j])

					for k := 0; k < len(grandChildsGr); k++ {
						//Search grandschilds
						if grandChildsGr[k].ParentID == childsGr[j].GroupID {
							pwcGrJson = append(pwcGrJson, grandChildsGr[k])
						}
					}
				}
			}
		}
		(*w).Header().Set("content-type", "application/json")
		err := json.NewEncoder(*w).Encode(pwcGrJson[:limit])
		if err != nil {
			log.Fatal("Cannot decode from JSON", err)
			return
		}
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

//Output group with GroupID == 0
func GetGroupTopParents(w http.ResponseWriter, req *http.Request) {
	var topParentsGroups []Group.Group
	for i := 0; i < len(Groups); i++ {
		if Groups[i].ParentID == 0 {
			topParentsGroups = append(topParentsGroups, Groups[i])
		}
	}
	w.Header().Set("content-type", "application/json")
	err := json.NewEncoder(w).Encode(topParentsGroups)
	if err != nil {
		log.Fatal("Cannot decode from JSON", err)
		return
	}
}

//Output group with GroupID == id
func GetGroupByID(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		fmt.Println("Cannot convert id to int")
		(w).WriteHeader(http.StatusBadRequest)
	}
	//search by id
	index := 0
	for i := 0; i < len(Groups); i++ {
		if Groups[i].GroupID == id {
			index = i
			break
		}
	}
	if index == len(Groups) {
		(w).WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(Groups[index])
	if err != nil {
		fmt.Println("Cannot decode from JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
}

//Output chids of group with GroupID == id
func GetGroupChildsByID(w http.ResponseWriter, req *http.Request) {
	var exist bool
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		fmt.Println("Cannot convert id to int")
		(w).WriteHeader(http.StatusBadRequest)
	}
	//Search chids of group with GroupID == id
	var childs []Group.Group
	for i := 0; i < len(Groups); i++ {
		if Groups[i].ParentID == id {
			childs = append(childs, Groups[i])
			exist = true
		}
	}
	//Encode and output found group
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(childs)
	if err != nil {
		log.Fatal("Cannot decode from JSON", err)
		return
	}
	if !exist {
		(w).WriteHeader(http.StatusNotFound)
	}
}

//Input new group
func PostNewGroup(w http.ResponseWriter, req *http.Request) {
	//Decode request body to Group type
	var postGr Group.Group
	err := json.NewDecoder(req.Body).Decode(&postGr)
	if err != nil {
		fmt.Println("Cannot decode from JSON", err)
		(w).WriteHeader(http.StatusBadRequest)
	}
	if postGr.GroupName == "" {
		(w).WriteHeader(http.StatusBadRequest)
		return
	}
	//ascending sort
	sort2.SliceStable(Groups, func(i, j int) bool {
		return Groups[i].GroupID < Groups[j].GroupID
	})
	//Input new group in Groups
	postGr.GroupID = Groups[len(Groups)-1].GroupID + 1
	Groups = append(Groups, postGr)
	(w).WriteHeader(http.StatusCreated)
}

//Change group with GroupID == id
func PutGroupByID(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		fmt.Println("Cannot convert id to int")
		(w).WriteHeader(http.StatusBadRequest)
	}
	//Decode request body to Group type
	var postGr Group.Group
	err = json.NewDecoder(req.Body).Decode(&postGr)
	if err != nil {
		log.Fatal("Cannot decode from JSON", err)
		return
	}
	if postGr.GroupName == "" {
		(w).WriteHeader(http.StatusBadRequest)
		return
	}
	//Search group with GroupID == id index
	index := 0
	for i := 0; i < len(Groups); i++ {
		if Groups[i].GroupID == id {
			index = i
			break
		}
	}
	if index == len(Groups) {
		(w).WriteHeader(http.StatusNotFound)
		return
	}
	//Encode and output found group
	Groups[index] = postGr
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(Groups[index])
	if err != nil {
		log.Fatal("Cannot decode from JSON", err)
		return
	}
}

//Delete group with GroupID = id and without children and tasks
func DeleteGroupByID(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		fmt.Println("Cannot convert id to int")
		(w).WriteHeader(http.StatusBadRequest)
	}
	//search index of element
	index := 0
	for i := 0; i < len(Groups); i++ {
		if Groups[i].GroupID == id {
			index = i
			break
		}
	}
	//does it have children
	badID := true
	for i := 0; i < len(Groups); i++ {
		if Groups[i].ParentID == id {
			badID = false
			break
		}
	}
	//does it have tasks
	if badID == true {
		for i := 0; i < len(Tasks); i++ {
			if Tasks[i].GroupID == id {
				badID = false
				break
			}
		}
	} else {
		(w).WriteHeader(http.StatusConflict)
		return
	}
	Groups = del(Groups, index)
}

func del(arr []Group.Group, n int) (outputArr []Group.Group) {
	for i := 0; i < len(arr); i++ {
		if i != n {
			outputArr = append(outputArr, arr[i])
		}
	}
	return outputArr
}

//Output tasks by sort, limit and type clarifications
func GetTasksSort(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	sort, srtOk := req.Form["sort"]
	limitstr, limOk := req.Form["limit"]
	typeOf, typeOk := req.Form["type"]

	var limit int
	if !limOk {
		limit = len(Tasks)
	} else {
		var err error
		limit, err = strconv.Atoi(limitstr[0])
		if err != nil {
			fmt.Println("Cannot convert limit to int")
			w.WriteHeader(http.StatusBadRequest)
		}
		if (limit > len(Tasks)) || (limit == 0) {
			limit = len(Tasks)
		}
	}

	getTasks := Tasks
	//Sort by type
	if typeOk {
		switch typeOf[0] {
		case "completed":
			getTasks = tasksTypeSort(Tasks, true)
			break
		case "working":
			getTasks = tasksTypeSort(Tasks, false)
			break
		default:
			break
		}
	}

	//Sort by sort type
	if srtOk {
		switch sort[0] {
		case "name":
			sort2.SliceStable(getTasks, func(i, j int) bool {
				return getTasks[i].Task < getTasks[j].Task
			})
			break
		case "group":
			sort2.SliceStable(getTasks, func(i, j int) bool {
				return getTasks[i].GroupID < getTasks[j].GroupID
			})
		default:
			break
		}
	}
	//Output
	w.Header().Set("content-type", "application/json")
	err := json.NewEncoder(w).Encode(getTasks[:limit])
	if err != nil {
		fmt.Println("Cannot decode from JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
}

//output array of completed or uncompleted tasks
func tasksTypeSort(tasks []Task.Task, typeof bool) (outputTasks []Task.Task) {
	for i := 0; i < len(tasks); i++ {
		if tasks[i].Completed == typeof {
			outputTasks = append(outputTasks, tasks[i])
		}
	}
	return outputTasks
}

//Input new task in Tasks
func PostNewTasks(w http.ResponseWriter, req *http.Request) {
	var postTask Task.Task
	err := json.NewDecoder(req.Body).Decode(&postTask)
	if err != nil {
		fmt.Println("Cannot decode from JSON", err)
		(w).WriteHeader(http.StatusBadRequest)
		return
	}
	if postTask.Task == "" {
		(w).WriteHeader(http.StatusBadRequest)
		return
	}

	//Check for group existence
	existGr := false
	for i := 0; i < len(Groups); i++ {
		existGr = existGr || (Groups[i].GroupID == postTask.GroupID)
	}
	if !existGr {
		fmt.Println("This GroupID do not exists", err)
		(w).WriteHeader(http.StatusNotFound)
		return
	}

	postTask.TaskID = taskNGrIDToHashToString5(postTask.Task, postTask.GroupID)

	//Chreck for matching task
	for i := 0; i < len(Tasks); i++ {
		if Tasks[i].TaskID == postTask.TaskID {
			fmt.Println("This task already exists", err)
			(w).WriteHeader(http.StatusConflict)
			return
		}
	}

	postTask.Completed = false
	postTask.CreatedAt = time.Now()
	Tasks = append(Tasks, postTask)

	//Output
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(Tasks[len(Tasks)-1])
	if err != nil {
		fmt.Println("Cannot decode from JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
	(w).WriteHeader(http.StatusCreated)
}

//Changing exist task
func PutTasksByID(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	//search by id
	index := len(Tasks)
	for i := 0; i < len(Tasks); i++ {
		if Tasks[i].TaskID == id {
			index = i
			break
		}
	}
	if index >= len(Tasks) {
		(w).WriteHeader(http.StatusNotFound)
		return
	}

	var postTask Task.Task
	err := json.NewDecoder(req.Body).Decode(&postTask)
	if err != nil {
		fmt.Println("Cannot decode from JSON", err)
		(w).WriteHeader(http.StatusBadRequest)
		return
	}
	if postTask.Task == "" {
		(w).WriteHeader(http.StatusBadRequest)
		return
	}

	//Check for group existence
	existGr := false
	for i := 0; i < len(Groups); i++ {
		existGr = existGr || (Groups[i].GroupID == postTask.GroupID)
	}
	if !existGr {
		fmt.Println("This GroupID do not exists", err)
		(w).WriteHeader(http.StatusNotFound)
		return
	}

	postTask.TaskID = taskNGrIDToHashToString5(postTask.Task, postTask.GroupID)

	//Check for matching task
	for i := 0; i < len(Tasks); i++ {
		if Tasks[i].TaskID == postTask.TaskID {
			fmt.Println("This task already exists", err)
			(w).WriteHeader(http.StatusConflict)
			return
		}
	}

	postTask.Completed = false
	postTask.CreatedAt = time.Now()

	Tasks[index] = postTask
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(Tasks[index])
	if err != nil {
		fmt.Println("Cannot decode from JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
	(w).WriteHeader(http.StatusCreated)
}

func GetTasksGroupByID(w http.ResponseWriter, req *http.Request) {
	//type := req.URL.Query().Get("type")
}

func PostTasksCompleteByID(w http.ResponseWriter, req *http.Request) {

}

func GetStatToday(w http.ResponseWriter, req *http.Request) {

}

func GetStatYesterday(w http.ResponseWriter, req *http.Request) {

}

func GetStatWeek(w http.ResponseWriter, req *http.Request) {

}

func GetStatMonth(w http.ResponseWriter, req *http.Request) {

}
