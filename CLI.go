package main

import (
	"WebCLI/Group"
	"WebCLI/Task"
	"context"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	sort2 "sort"
	"strconv"
	"time"
)

var Tasks []Task.Task
var Groups []Group.Group

func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

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
	router.HandleFunc("/tasks/group/{id}", GetTasksGroupByID).Methods(http.MethodGet)
	router.HandleFunc("/tasks/{id}", PostTasksCompleteByID).Methods(http.MethodPost)
	router.HandleFunc("/stat/today", GetStatToday).Methods(http.MethodGet)
	router.HandleFunc("/stat/yesterday", GetStatYesterday).Methods(http.MethodGet)
	router.HandleFunc("/stat/week", GetStatWeek).Methods(http.MethodGet)
	router.HandleFunc("/stat/month", GetStatMonth).Methods(http.MethodGet)

	//Graceful shutdown
	srv := &http.Server{
		Addr: "0.0.0.0:8181",
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router, // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	Task.JsonTaskOutput(Tasks)
	Group.JsonGroupOutput(Groups)
	log.Println("shutting down")
	os.Exit(0)
}

func taskNGrIDToHashToString6(task string, grID int) (str string) {
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

	//sorted or unsorted output
	if !srtOk {
		unJsonedGr := Groups[:limit]
		w.Header().Set("content-type", "application/json")
		err := json.NewEncoder(w).Encode(unJsonedGr)
		if err != nil {
			fmt.Println("Cannot encode to JSON", err)
			(w).WriteHeader(http.StatusConflict)
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
			fmt.Println("Cannot encode to JSON", err)
			(*w).WriteHeader(http.StatusConflict)
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
			fmt.Println("Cannot encode to JSON", err)
			(*w).WriteHeader(http.StatusConflict)
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
			fmt.Println("Cannot encode to JSON", err)
			(*w).WriteHeader(http.StatusConflict)
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
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
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
		fmt.Println("Cannot encode to JSON", err)
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
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
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
	err = json.NewEncoder(w).Encode(Groups[len(Groups)-1])
	if err != nil {
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
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
		fmt.Println("Cannot decode from JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
	if postGr.GroupName == "" {
		(w).WriteHeader(http.StatusBadRequest)
		return
	}
	//Search group with GroupID == id index
	index := len(Groups)
	for i := 0; i < len(Groups); i++ {
		if Groups[i].GroupID == id {
			index = i
			break
		}
	}
	if index >= len(Groups) {
		(w).WriteHeader(http.StatusNotFound)
		return
	}
	//Encode and output found group
	postGr.GroupID = id
	Groups[index] = postGr
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(Groups[index])
	if err != nil {
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
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
		case "all":
			break
		default:
			fmt.Println("Invalid type argument")
			(w).WriteHeader(http.StatusBadRequest)
			return
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
			fmt.Println("Invalid sort argument")
			(w).WriteHeader(http.StatusBadRequest)
			return
		}
	}
	//Output
	w.Header().Set("content-type", "application/json")
	err := json.NewEncoder(w).Encode(getTasks[:limit])
	if err != nil {
		fmt.Println("Cannot encode to JSON", err)
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
	var postInputTask Task.Task
	err := json.NewDecoder(req.Body).Decode(&postInputTask)
	if err != nil {
		fmt.Println("Cannot decode from JSON", err)
		(w).WriteHeader(http.StatusBadRequest)
		return
	}
	var postTask Task.Task
	postTask.Task = postInputTask.Task
	postTask.GroupID = postInputTask.GroupID
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

	postTask.TaskID = taskNGrIDToHashToString6(postTask.Task, postTask.GroupID)

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
		fmt.Println("Cannot encode to JSON", err)
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

	var postInputTask Task.Task
	err := json.NewDecoder(req.Body).Decode(&postInputTask)
	if err != nil {
		fmt.Println("Cannot decode from JSON", err)
		(w).WriteHeader(http.StatusBadRequest)
		return
	}
	var postTask Task.Task
	postTask.Task = postInputTask.Task
	postTask.GroupID = postInputTask.GroupID
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

	postTask.TaskID = taskNGrIDToHashToString6(postTask.Task, postTask.GroupID)

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
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
	(w).WriteHeader(http.StatusCreated)
}

//Output tasks of group with GroupID == id
func GetTasksGroupByID(w http.ResponseWriter, req *http.Request) {
	typeOf, typeOk := req.Form["type"]
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		fmt.Println("Cannot convert id to int")
		(w).WriteHeader(http.StatusBadRequest)
	}

	var sortedTasks []Task.Task
	for i := 0; i < len(Tasks); i++ {
		if Tasks[i].GroupID == id {
			sortedTasks = append(sortedTasks, Tasks[i])
		}
	}
	if typeOk {
		switch typeOf[0] {
		case "completed":
			sortedTasks = tasksTypeSort(Tasks, true)
			break
		case "working":
			sortedTasks = tasksTypeSort(Tasks, false)
			break
		case "all":
			break
		default:
			fmt.Println("Invalid type")
			(w).WriteHeader(http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(sortedTasks)
	if err != nil {
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
}

//Changeing complete status of task
func PostTasksCompleteByID(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	finish, finOk := req.Form["finished"]
	vars := mux.Vars(req)
	id := vars["id"]

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

	if !finOk {
		fmt.Println("Missing request argument")
		(w).WriteHeader(http.StatusBadRequest)
	}

	switch finish[0] {
	case "true":
		Tasks[index].Completed = true
		Tasks[index].CompletedAt = time.Now()
		break
	case "false":
		Tasks[index].Completed = false
		var nilTime time.Time
		Tasks[index].CompletedAt = nilTime
		break
	default:
		fmt.Println("Invalid request argument")
		(w).WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("content-type", "application/json")
	err := json.NewEncoder(w).Encode(Tasks[index])
	if err != nil {
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
}

//Output tasks completed today
func GetStatToday(w http.ResponseWriter, req *http.Request) {
	tasksCount := make(map[string]int)
	tasksCount["completed"] = 0
	tasksCount["created"] = 0
	timeNow := time.Now()
	nowYear, nowMonth, nowDay := timeNow.Date()

	for i := 0; i < len(Tasks); i++ {
		crYear, crMonth, crDay := Tasks[i].CreatedAt.Date()
		if (crYear == nowYear) && (crMonth == nowMonth) && (crDay == nowDay) {
			tasksCount["created"]++
		}
		if Tasks[i].Completed {
			comYear, comMonth, comDay := Tasks[i].CompletedAt.Date()
			if (comYear == nowYear) && (comMonth == nowMonth) && (comDay == nowDay) {
				tasksCount["completed"]++
			}
		}
	}
	w.Header().Set("content-type", "application/json")
	err := json.NewEncoder(w).Encode(tasksCount)
	if err != nil {
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
}

//Output tasks completed yesterday
func GetStatYesterday(w http.ResponseWriter, req *http.Request) {
	tasksCount := make(map[string]int)
	tasksCount["completed"] = 0
	tasksCount["created"] = 0
	timeNow := time.Now().AddDate(0, 0, -1)
	nowYear, nowMonth, nowDay := timeNow.Date()

	for i := 0; i < len(Tasks); i++ {
		crYear, crMonth, crDay := Tasks[i].CreatedAt.Date()
		if (crYear == nowYear) && (crMonth == nowMonth) && (crDay == nowDay) {
			tasksCount["created"]++
		}
		if Tasks[i].Completed {
			comYear, comMonth, comDay := Tasks[i].CompletedAt.Date()
			if (comYear == nowYear) && (comMonth == nowMonth) && (comDay == nowDay) {
				tasksCount["completed"]++
			}
		}
	}
	w.Header().Set("content-type", "application/json")
	err := json.NewEncoder(w).Encode(tasksCount)
	if err != nil {
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
}

//Output tasks completed within a week
func GetStatWeek(w http.ResponseWriter, req *http.Request) {
	tasksCount := make(map[string]int)
	tasksCount["completed"] = 0
	tasksCount["created"] = 0
	timeNow := time.Now()
	timeWeekAgo := time.Now().AddDate(0, 0, -7)
	weekDur := timeNow.Sub(timeWeekAgo)

	for i := 0; i < len(Tasks); i++ {
		if timeNow.Sub(Tasks[i].CreatedAt) <= weekDur {
			tasksCount["created"]++
		}
		if Tasks[i].Completed {
			if timeNow.Sub(Tasks[i].CompletedAt) <= weekDur {
				tasksCount["completed"]++
			}
		}
	}
	w.Header().Set("content-type", "application/json")
	err := json.NewEncoder(w).Encode(tasksCount)
	if err != nil {
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
}

//Output tasks completed within a month
func GetStatMonth(w http.ResponseWriter, req *http.Request) {
	tasksCount := make(map[string]int)
	tasksCount["completed"] = 0
	tasksCount["created"] = 0
	timeNow := time.Now()
	timeMonthAgo := time.Now().AddDate(0, -1, 0)
	monthDur := timeNow.Sub(timeMonthAgo)

	for i := 0; i < len(Tasks); i++ {
		if timeNow.Sub(Tasks[i].CreatedAt) <= monthDur {
			tasksCount["created"]++
		}
		if Tasks[i].Completed {
			if timeNow.Sub(Tasks[i].CompletedAt) <= monthDur {
				tasksCount["completed"]++
			}
		}
	}
	w.Header().Set("content-type", "application/json")
	err := json.NewEncoder(w).Encode(tasksCount)
	if err != nil {
		fmt.Println("Cannot encode to JSON", err)
		(w).WriteHeader(http.StatusConflict)
		return
	}
}
