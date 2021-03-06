package main

import (
	"WebCLI/Group"
	"WebCLI/Service"
	"WebCLI/Task"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"io"
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

//TODO:default values read from configuration file
var defaultGrLimit, defaultTaskLimit, defaultTaskGr int
var port, defaultTaskComplete, defaultTaskType, defaultTaskSort string

func main() {
	vp := viper.New()
	port, defaultGrLimit, _, _, _, _, _ = configToDefaults(vp)
	_, _, defaultTaskComplete, defaultTaskLimit, defaultTaskGr, defaultTaskType, defaultTaskSort = configToDefaults(vp)

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
	router.HandleFunc("/tasks/group/{id}", GetTasksByGroupID).Methods(http.MethodGet)
	router.HandleFunc("/tasks/{id}", PostTasksCompleteByID).Methods(http.MethodPost)
	router.HandleFunc("/stat/today", GetStatToday).Methods(http.MethodGet)
	router.HandleFunc("/stat/yesterday", GetStatYesterday).Methods(http.MethodGet)
	router.HandleFunc("/stat/week", GetStatWeek).Methods(http.MethodGet)
	router.HandleFunc("/stat/month", GetStatMonth).Methods(http.MethodGet)

	//Graceful shutdown
	srv := &http.Server{
		Addr: "0.0.0.0:" + port,
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

//TODO:Read configuration file and input in variables
func configToDefaults(vp *viper.Viper) (p string, dgl int, dtc string, dtg int, dtl int, dtt string, dts string) {
	vp.SetConfigFile("Service\\Config.toml")
	err := vp.ReadInConfig()
	if err != nil {
		Service.ErrExecLog(http.StatusInternalServerError, 0, err, "Configuration file read error")
	}
	p = vp.GetString("Application.port")
	dgl = vp.GetInt("Groups.default_limit")
	dtc = vp.GetString("Tasks.default_complete")
	dtg = vp.GetInt("Tasks.default_group")
	dtl = vp.GetInt("Tasks.default_limit")
	dtt = vp.GetString("Tasks.default_type")
	dts = vp.GetString("Tasks.default_sort")
	return p, dgl, dtc, dtg, dtl, dtt, dts
}

//TODO:Hashing
func taskNGrIDToHashToString6(task string, grID int) (str string) {
	task += strconv.Itoa(grID)
	hsh := md5.Sum([]byte(task))
	str = fmt.Sprintf("%x", hsh)
	return str[:6]
}

//TODO:Output groups at response
func GetGroups(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.FuncWorkLog(strconv.Itoa(len(Groups)), "real size of group slice")
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
	sort, srtOk := req.Form["sort"]
	limitstr, limOk := req.Form["limit"]
	var limit int
	if !limOk {
		Service.WarnLog("GetGroups", "limit")
		limit = defaultGrLimit
	} else {
		limit, _ = strconv.Atoi(limitstr[0])
		if (limit > len(Groups)) || (limit == 0) {
			limit = len(Groups)
		}
	}

	//sorted or unsorted output
	sort2.SliceStable(Groups, func(i, j int) bool { return Groups[i].GroupName < Groups[j].GroupName })
	if !srtOk {
		Service.WarnLog("GetGroups", "sort")
		unJsonedGr := Groups[:limit]
		w.Header().Set("content-type", "application/json")
		err := json.NewEncoder(w).Encode(unJsonedGr)
		if err != nil {
			msg = "Cannot encode to JSON"
			statCode = http.StatusInternalServerError
			(w).WriteHeader(statCode)
			return
		}
	} else {
		GetGroupsSort(&w, req, sort[0], limit, &statCode, &msg, &err)
	}
}

//TODO:Sorting groups by GetGroup request parameters
func GetGroupsSort(w *http.ResponseWriter, req *http.Request, sort string, limit int, statCode *int, msg *string, err *error) {

	//unmarshall json file to groups' slice and ascending sort by name
	var unJsonedGr, parentsGr, childsGr, childGr, grandChildsGr []Group.Group
	unJsonedGr = Groups

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
			*msg = "Cannot encode to JSON"
			*statCode = http.StatusInternalServerError
			(*w).WriteHeader(*statCode)
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
			*msg = "Cannot encode to JSON"
			*statCode = http.StatusInternalServerError
			(*w).WriteHeader(*statCode)
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
			*msg = "Cannot encode to JSON"
			*statCode = http.StatusInternalServerError
			(*w).WriteHeader(*statCode)
			return
		}
		break
	default:
		*err = errors.New("")
		*msg = "Invalid arguments"
		*statCode = http.StatusBadRequest
		(*w).WriteHeader(*statCode)
		break
	}
}

func grContain(arrGr []Group.Group, contGr Group.Group) (result bool) {
	for i := 0; i < len(arrGr); i++ {
		result = result || arrGr[i].GroupID == contGr.ParentID
	}
	return result
}

//TODO:Output group with GroupID == 0
func GetGroupTopParents(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
	var topParentsGroups []Group.Group
	for i := 0; i < len(Groups); i++ {
		if Groups[i].ParentID == 0 {
			topParentsGroups = append(topParentsGroups, Groups[i])
		}
	}
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(topParentsGroups)
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
}

//TODO:Output group with GroupID == id
func GetGroupByID(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		msg = "Cannot convert id to int"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
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
		msg = "Group with that id does not exist"
		statCode = http.StatusNotFound
		(w).WriteHeader(statCode)
		return
	}
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(Groups[index])
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
}

//TODO:Output children of group with GroupID == id
func GetGroupChildsByID(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
	var exist bool
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		msg = "Cannot convert id to int"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
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
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
	if !exist {
		err = errors.New("")
		msg = "Group children not found"
		statCode = http.StatusNotFound
		(w).WriteHeader(statCode)
		return
	}
}

//TODO:Input new group
func PostNewGroup(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	//Decode request body to Group type
	var postGr Group.Group
	var buf bytes.Buffer
	bodyCopy := io.TeeReader(req.Body, &buf)
	Service.ReqInfoLog(req.Method, req.URL, req.Form, bodyCopy, "")
	err = json.NewDecoder(&buf).Decode(&postGr)
	if err != nil {
		msg = "Cannot decode from JSON"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}
	if postGr.GroupName == "" {
		err = errors.New("")
		msg = "Field group_name cannot be empty"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}
	//ascending sort
	sort2.SliceStable(Groups, func(i, j int) bool {
		return Groups[i].GroupID < Groups[j].GroupID
	})
	//Input new group in Groups
	postGr.GroupID = Groups[len(Groups)-1].GroupID + 1
	Groups = append(Groups, postGr)
	err = json.NewEncoder(w).Encode(Groups[len(Groups)-1])
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
	(w).WriteHeader(http.StatusCreated)
}

//TODO:Change group with GroupID == id
func PutGroupByID(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	var buf bytes.Buffer
	bodyCopy := io.TeeReader(req.Body, &buf)
	Service.ReqInfoLog(req.Method, req.URL, req.Form, bodyCopy, "")
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		msg = "Cannot convert id to int"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}
	//Decode request body to Group type
	var postGr Group.Group
	err = json.NewDecoder(&buf).Decode(&postGr)
	if err != nil {
		msg = "Cannot decode from JSON"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}
	if postGr.GroupName == "" {
		err = errors.New("")
		msg = "Field group_name cannot be empty"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
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
	if index == len(Groups) {
		err = errors.New("")
		msg = "Group with that id does not exist"
		statCode = http.StatusNotFound
		(w).WriteHeader(statCode)
		return
	}
	//Encode and output found group
	postGr.GroupID = id
	Groups[index] = postGr
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(Groups[index])
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
	(w).WriteHeader(http.StatusCreated)
}

//TODO:Delete group with GroupID = id and without children and tasks
func DeleteGroupByID(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		msg = "Cannot convert id to int"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
	}
	//search index of element
	index := len(Groups)
	for i := 0; i < len(Groups); i++ {
		if Groups[i].GroupID == id {
			index = i
			break
		}
	}

	if index == len(Groups) {
		err = errors.New("")
		msg = "Group with that id does not exist"
		statCode = http.StatusNotFound
		(w).WriteHeader(statCode)
		return
	}
	//does it have children
	for i := 0; i < len(Groups); i++ {
		if Groups[i].ParentID == id {
			msg = "Group have the children"
			statCode = http.StatusBadRequest
			(w).WriteHeader(statCode)
			return
		}
	}
	//does it have tasks
	for i := 0; i < len(Tasks); i++ {
		if Tasks[i].GroupID == id {
			msg = "Group have the tasks"
			statCode = http.StatusBadRequest
			(w).WriteHeader(statCode)
			return
		}
		Groups = del(Groups, index)
	}
	statCode = http.StatusOK
	(w).WriteHeader(statCode)
}

func del(arr []Group.Group, n int) (outputArr []Group.Group) {
	for i := 0; i < len(arr); i++ {
		if i != n {
			outputArr = append(outputArr, arr[i])
		}
	}
	return outputArr
}

//TODO:Output tasks by sort, limit and type clarifications
func GetTasksSort(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.FuncWorkLog(strconv.Itoa(len(Tasks)), "real size of task slice")
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
	sort, srtOk := req.Form["sort"]
	limitstr, limOk := req.Form["limit"]
	typeOf, typeOk := req.Form["type"]

	var limit int
	if !limOk {
		Service.WarnLog("GetTasksSort", "limit")
		limit = defaultTaskLimit
	} else {
		var err error
		limit, err = strconv.Atoi(limitstr[0])
		if err != nil {
			msg = "Cannot convert limit to int"
			statCode = http.StatusBadRequest
			(w).WriteHeader(statCode)
			return
		}
		if (limit > len(Tasks)) || (limit == 0) {
			limit = len(Tasks)
		}
	}

	getTasks := Tasks
	//Sort by type
	if !typeOk {
		Service.WarnLog("GetTasksSort", "type")
		typeOf[0] = defaultTaskType
	}
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
		err = errors.New("")
		msg = "Invalid type argument"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}

	//Sort by sort type
	if !srtOk {
		Service.WarnLog("GetTasksSort", "sort")
		sort[0] = defaultTaskSort
	}
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
		err = errors.New("")
		msg = "Invalid sort argument"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}

	//Output
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(getTasks[:limit])
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
}

//TODO:Output array of completed or uncompleted tasks
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
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	var buf bytes.Buffer
	bodyCopy := io.TeeReader(req.Body, &buf)
	Service.ReqInfoLog(req.Method, req.URL, req.Form, bodyCopy, "")
	var postInputTask Task.Task
	err = json.NewDecoder(&buf).Decode(&postInputTask)
	if err != nil {
		msg = "Cannot decode from JSON"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}
	var postTask Task.Task
	postTask.Task = postInputTask.Task
	postTask.GroupID = postInputTask.GroupID
	if postTask.GroupID == 0 {
		Service.WarnLog("PostNewTasks", "GroupId")
		postTask.GroupID = defaultTaskGr
	}
	if postTask.Task == "" {
		err = errors.New("")
		msg = "Field group_name cannot be empty"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}

	//Check for group existence
	existGr := false
	for i := 0; i < len(Groups); i++ {
		existGr = existGr || (Groups[i].GroupID == postTask.GroupID)
	}
	if !existGr {
		err = errors.New("")
		msg = "This GroupID do not exists"
		statCode = http.StatusNotFound
		(w).WriteHeader(statCode)
		return
	}

	postTask.TaskID = taskNGrIDToHashToString6(postTask.Task, postTask.GroupID)

	//Check for matching task
	for i := 0; i < len(Tasks); i++ {
		if Tasks[i].TaskID == postTask.TaskID {
			err = errors.New("")
			msg = "This task already exists"
			statCode = http.StatusBadRequest
			(w).WriteHeader(statCode)
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
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
	(w).WriteHeader(http.StatusCreated)
}

//TODO:Changing exist task
func PutTasksByID(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	var buf bytes.Buffer
	bodyCopy := io.TeeReader(req.Body, &buf)
	Service.ReqInfoLog(req.Method, req.URL, req.Form, bodyCopy, "")
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
	if index == len(Tasks) {
		err = errors.New("")
		msg = "Task with that id does not exist"
		statCode = http.StatusNotFound
		(w).WriteHeader(statCode)
		return
	}

	var postInputTask Task.Task
	err = json.NewDecoder(&buf).Decode(&postInputTask)
	if err != nil {
		msg = "Cannot decode from JSON"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}
	var postTask Task.Task
	postTask.Task = postInputTask.Task
	postTask.GroupID = postInputTask.GroupID
	if postTask.Task == "" {
		err = errors.New("")
		msg = "Field task cannot be empty"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}
	if postTask.GroupID == 0 {
		Service.WarnLog("PostNewTasks", "GroupId")
		postTask.GroupID = defaultTaskGr
	}

	//Check for group existence
	existGr := false
	for i := 0; i < len(Groups); i++ {
		existGr = existGr || (Groups[i].GroupID == postTask.GroupID)
	}
	if !existGr {
		err = errors.New("")
		msg = "Group with this GroupID do not exists"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}

	postTask.TaskID = taskNGrIDToHashToString6(postTask.Task, postTask.GroupID)

	//Check for matching task
	for i := 0; i < len(Tasks); i++ {
		if Tasks[i].TaskID == postTask.TaskID {
			err = errors.New("")
			msg = "This task already exists"
			statCode = http.StatusBadRequest
			(w).WriteHeader(statCode)
			return
		}
	}

	postTask.Completed = false
	postTask.CreatedAt = time.Now()

	Tasks[index] = postTask
	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(Tasks[index])
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
	(w).WriteHeader(http.StatusCreated)
}

//TODO:Output tasks of group with GroupID == id
func GetTasksByGroupID(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
	typeOf, typeOk := req.Form["type"]
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		msg = "Cannot convert id to int"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
	}

	var sortedTasks []Task.Task
	for i := 0; i < len(Tasks); i++ {
		if Tasks[i].GroupID == id {
			sortedTasks = append(sortedTasks, Tasks[i])
		}
	}

	if !typeOk {
		Service.WarnLog("GetTasksSort", "type")
		typeOf[0] = defaultTaskType
	}
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
		err = errors.New("")
		msg = "Invalid type argument"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}

	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(sortedTasks)
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
}

//TODO:Changeing complete status of task
func PostTasksCompleteByID(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")

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
	if index == len(Tasks) {
		err = errors.New("")
		msg = "Task with that id does not exist"
		statCode = http.StatusNotFound
		(w).WriteHeader(statCode)
		return
	}

	if !finOk {
		Service.WarnLog("GetTasksSort", "type")
		finish[0] = defaultTaskComplete
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
		err = errors.New("")
		msg = "Invalid request argument"
		statCode = http.StatusBadRequest
		(w).WriteHeader(statCode)
		return
	}

	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(Tasks[index])
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
}

//TODO:Output tasks completed today
func GetStatToday(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
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

	tmp, _ := json.Marshal(tasksCount)
	logOut := string(tmp)
	Service.FuncWorkLog(logOut, "called in function GetStatToday")

	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(tasksCount)
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
}

//TODO:Output tasks completed yesterday
func GetStatYesterday(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
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

	tmp, _ := json.Marshal(tasksCount)
	logOut := string(tmp)
	Service.FuncWorkLog(logOut, "called in function GetStatYesterday")

	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(tasksCount)
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
}

//TODO:Output tasks completed within a week
func GetStatWeek(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
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

	tmp, _ := json.Marshal(tasksCount)
	logOut := string(tmp)
	Service.FuncWorkLog(logOut, "called in function GetStatWeek")

	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(tasksCount)
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
}

//TODO:Output tasks completed within a month
func GetStatMonth(w http.ResponseWriter, req *http.Request) {
	var err error
	var msg string
	statCode := http.StatusOK
	var startTime, endTime time.Time
	startTime = time.Now()
	defer func() {
		endTime = time.Now()
		Service.ErrExecLog(statCode, endTime.Sub(startTime), err, msg)
	}()

	req.ParseForm()
	Service.ReqInfoLog(req.Method, req.URL, req.Form, req.Body, "")
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

	tmp, _ := json.Marshal(tasksCount)
	logOut := string(tmp)
	Service.FuncWorkLog(logOut, "called in function GetStatMonth")

	w.Header().Set("content-type", "application/json")
	err = json.NewEncoder(w).Encode(tasksCount)
	if err != nil {
		msg = "Cannot encode to JSON"
		statCode = http.StatusInternalServerError
		(w).WriteHeader(statCode)
		return
	}
}
