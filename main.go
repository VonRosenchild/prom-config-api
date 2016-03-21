package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/julienschmidt/httprouter"
	"github.com/percona/prom-config-api/prom"
)

var (
	LISTEN      = ":9003"
	PROMDIR     = "/opt/prometheus"
	OS_PORTS    = []string{"9100"}
	MYSQL_PORTS = []string{"9104", "9105", "9106"}
)

var tf *prom.TargetsFile

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
	targets := map[string][]prom.Target{
		"os":    make([]prom.Target, len(OS_PORTS)),
		"mysql": make([]prom.Target, len(MYSQL_PORTS)),
	}
	for i, port := range OS_PORTS {
		targets["os"][i] = prom.Target{
			Port:     port,
			Filename: path.Join(PROMDIR, "targets_"+port+".yml"),
		}
	}
	for i, port := range MYSQL_PORTS {
		targets["mysql"][i] = prom.Target{
			Port:     port,
			Filename: path.Join(PROMDIR, "targets_"+port+".yml"),
		}
	}
	tf = prom.NewTargetsFile(hostsFile, targets)

	router := httprouter.New()
	router.GET("/hosts", list)
	router.POST("/hosts/:type", add)
	router.DELETE("/hosts/:type/:alias", remove)

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

func add(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ErrorResponse(w, err)
		return
	}
	if len(body) == 0 {
		JSONResponse(w, http.StatusBadRequest, nil)
		return
	}
	var host prom.Host
	if err := json.Unmarshal(body, &host); err != nil {
		ErrorResponse(w, err)
		return
	}

	hostType := p.ByName("type")

	if err := tf.Add(hostType, host); err != nil {
		ErrorResponse(w, err)
	} else {
		JSONResponse(w, http.StatusCreated, nil)
	}
	log.Printf("Added %s %+v", hostType, host)
}

func remove(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	hostType := p.ByName("type")
	alias := p.ByName("alias")
	err := tf.Remove(hostType, alias)
	if err != nil {
		if err == prom.ErrHostNotFound {
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
	e := prom.Error{
		Error: err.Error(),
	}
	if err := json.NewEncoder(w).Encode(e); err != nil {
		panic(err)
	}
}
