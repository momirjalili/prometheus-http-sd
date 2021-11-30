package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/momirjalili/httpsd/internal/httpsd"
	bolt "go.etcd.io/bbolt"
)

type SDServer struct {
	store *httpsd.TargetStore
}

func NewSDServer(db *bolt.DB) *SDServer {
	store := httpsd.New(db)
	return &SDServer{store: store}
}

// renderJSON renders 'v' as JSON and writes it as a response into w.
func renderJSON(w http.ResponseWriter, v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// GET /api/v1/target/    return targets list
func (sd *SDServer) GetAllTargetGroupsHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("getting all target groups\n")
	allTGs, err := sd.store.GetAllTargetGroups()
	if err != nil {
		fmt.Printf("error getting all targets")
	}
	renderJSON(w, allTGs)
}

//createTargetGroupHandler POST /api/v1/target/     creates a new target group
func (sd *SDServer) CreateTargetGroupHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling target group create at %s\n", req.URL.Path)
	fmt.Printf("body is %s \n", req.Body)
	// Enforce a JSON Content-Type.
	contentType := req.Header.Get("Content-Type")
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if mediatype != "application/json" {
		http.Error(w, "expect application/json Content-Type", http.StatusUnsupportedMediaType)
		return
	}
	dec := json.NewDecoder(req.Body)
	var tg httpsd.TargetGroup
	if err := dec.Decode(&tg); err != nil {
		fmt.Printf("error decoding %s \n", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("decoded target group  is %v", tg)
	err = sd.store.CreateTargetGroup(&tg)

	if err != nil {
		fmt.Printf("error on storing targetgroup %s\n ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (sd *SDServer) GetTargetGroupHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("getting target group")
	id, _ := strconv.Atoi(mux.Vars(req)["id"])
	tg, err := sd.store.GetTargetGroup(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	renderJSON(w, tg)
}

func (sd *SDServer) PutTargetHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("adding target to target group")
	id, _ := strconv.Atoi(mux.Vars(req)["id"])
	tg, err := sd.store.GetTargetGroup(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	dec := json.NewDecoder(req.Body)
	var tat httpsd.TargetGroup
	if err := dec.Decode(&tat); err != nil {
		fmt.Printf("error decoding %s \n", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("sent data is tat: %+v", tat)
	// updating targets
	// ntg := httpsd.NewSet(tg.Targets)
	// ntg.Add(tat.Targets)
	// tg.Targets = ntg.Array()

	// updating labels
	for k, v := range tat.Labels {
		_, ok := tg.Labels[k]
		if ok {
			fmt.Printf("label exists\n")
			http.Error(w, "label already exists. to update use PATCH", http.StatusBadRequest)
		}
		tg.Labels[k] = v
	}
	utg, err := sd.store.UpdateTargetGroup(tg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderJSON(w, utg)
}

// PATCH  /api/v1/target/<target_group_id>/label/<label_key>     # updates a label in a target group
func (sd *SDServer) PatchTargetGroupLabelHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("patching labels from target group")
	id, _ := strconv.Atoi(mux.Vars(req)["id"])
	tg, err := sd.store.GetTargetGroup(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// updating labels
	label := mux.Vars(req)["label_key"]
	_, ok := tg.Labels[label]
	if !ok {
		fmt.Printf("label not exists\n")
		http.Error(w, "label does not exists.", http.StatusNotFound)
	}
	v, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tg.Labels[label] = v
	utg, err := sd.store.UpdateTargetGroup(tg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderJSON(w, utg)
}

// DELTE  /api/v1/target/<target_group_id>/label/<label_key>     # deletes a label in a target group
func (sd *SDServer) DeleteTargetGroupLabelHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("deleting label from target group")
	id, _ := strconv.Atoi(mux.Vars(req)["id"])
	tg, err := sd.store.GetTargetGroup(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// updating labels
	label := mux.Vars(req)["label_key"]
	delete(tg.Labels, label)
	utg, err := sd.store.UpdateTargetGroup(tg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderJSON(w, utg)
}