package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/julienschmidt/httprouter"
)

var (
	LISTEN  = ":9003"
	PROMDIR = "/opt/prometheus"
	PORTS   = []string{"9100", "9104", "9105", "9106"}
)

var tf *TargetsFile

func init() {
	if listen := os.Getenv("LISTEN"); listen != "" {
		LISTEN = listen
	}
	if promdir := os.Getenv("PROMDIR"); promdir != "" {
		PROMDIR = promdir
	}
}

func main() {
	if _, err := os.Stat(PROMDIR); err != nil {
		log.Fatal(err)
	}

	hostsFile := path.Join(PROMDIR, "hosts.yml")
	if _, err := os.Stat(hostsFile); err != nil {
		f, err := os.Create(hostsFile)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	}
	targets := make([]Target, len(PORTS))
	for i, port := range PORTS {
		targets[i] = Target{
			Port:     port,
			Filename: path.Join(PROMDIR, "targets_"+port+".yml"),
		}
	}
	tf = NewTargetsFile(hostsFile, targets)

	router := httprouter.New()
	router.GET("/hosts", list)
	router.POST("/hosts", add)
	router.DELETE("/hosts/:alias", remove)

	log.Fatal(http.ListenAndServe(LISTEN, router))
}

func list(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	hosts, err := tf.List()
	if err != nil {
		ErrorResponse(w, err)
	} else {
		JSONResponse(w, http.StatusOK, hosts)
	}
}

func add(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ErrorResponse(w, err)
		return
	}
	if len(body) == 0 {
		JSONResponse(w, http.StatusBadRequest, nil)
		return
	}
	var host Host
	if err := json.Unmarshal(body, &host); err != nil {
		ErrorResponse(w, err)
		return
	}

	if err := tf.Add(host); err != nil {
		ErrorResponse(w, err)
	} else {
		JSONResponse(w, http.StatusCreated, nil)
	}
	log.Printf("Added %+v", host)
}

func remove(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	alias := p.ByName("alias")
	err := tf.Remove(alias)
	if err != nil {
		if err == ErrHostNotFound {
			http.NotFound(w, r)
		} else {
			ErrorResponse(w, err)
		}
	} else {
		JSONResponse(w, http.StatusOK, nil)
	}
	log.Printf("Removed %s", alias)
}

func WriteAccessControlHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func JSONResponse(w http.ResponseWriter, statusCode int, v interface{}) {
	WriteAccessControlHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			panic(err)
		}
	}
}

func ErrorResponse(w http.ResponseWriter, err error) {
	WriteAccessControlHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(500)
	e := Error{
		Error: err.Error(),
	}
	if err := json.NewEncoder(w).Encode(e); err != nil {
		panic(err)
	}
}
