package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/julienschmidt/httprouter"
	"github.com/percona/platform/proto"
	"github.com/percona/prom-config-api/prom"
)

var (
	DEFAULT_flagListen = ":" + proto.DEFAULT_PROM_CONFIG_API_PORT
	DEFAULT_BASEDIR    = "/opt/prometheus"
)

var (
	flagBasedir string
	flagListen  string
	flagVersion bool
)

func init() {
	flag.StringVar(&flagBasedir, "basedir", DEFAULT_BASEDIR, "Dir to use for hosts.yml and target files")
	flag.StringVar(&flagListen, "listen", DEFAULT_flagListen, "IP:port to listen on")
	flag.BoolVar(&flagVersion, "version", false, "Print version")
	flag.Parse()
}

var (
	VERSION     = "1.0.0"
	OS_PORTS    = []string{"9100"}
	MYSQL_PORTS = []string{"9104", "9105", "9106"}
	tf          *prom.TargetsFile
)

func main() {
	if flagVersion {
		fmt.Printf("prom-config-api %s\n", VERSION)
		os.Exit(0)
	}

	log.Println("INFO: prom-config-api", VERSION, "basedir", flagBasedir)

	if _, err := os.Stat(flagBasedir); err != nil {
		log.Fatal(err)
	}

	hostsFile := path.Join(flagBasedir, "hosts.yml")
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
			Filename: path.Join(flagBasedir, "targets_"+port+".yml"),
		}
	}
	for i, port := range MYSQL_PORTS {
		targets["mysql"][i] = prom.Target{
			Port:     port,
			Filename: path.Join(flagBasedir, "targets_"+port+".yml"),
		}
	}
	tf = prom.NewTargetsFile(hostsFile, targets)

	router := httprouter.New()
	router.GET("/hosts", list)
	router.POST("/hosts/:type", add)
	router.DELETE("/hosts/:type/:alias", remove)

	log.Printf("INFO: listening on %s...", flagListen)
	log.Fatal(http.ListenAndServe(flagListen, router))
}

func list(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	hosts, err := tf.List()
	if err != nil {
		log.Println("ERROR: list: tf.List:", err)
		proto.ErrorResponse(w, err)
	} else {
		proto.JSONResponse(w, http.StatusOK, hosts)
	}
}

func add(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("ERROR: add: ioutil.ReadAll:", err)
		proto.ErrorResponse(w, err)
		return
	}
	if len(body) == 0 {
		proto.JSONResponse(w, http.StatusBadRequest, nil)
		return
	}
	var host proto.Host
	if err := json.Unmarshal(body, &host); err != nil {
		log.Println("ERROR: add: json.Unmarshal:", err)
		proto.ErrorResponse(w, err)
		return
	}

	hostType := p.ByName("type")

	if err := tf.Add(hostType, host); err != nil {
		log.Println("ERROR: add: tf.Add:", err)
		proto.ErrorResponse(w, err)
	} else {
		log.Printf("INFO: added %s %+v", hostType, host)
		proto.JSONResponse(w, http.StatusCreated, nil)
	}
}

func remove(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	hostType := p.ByName("type")
	alias := p.ByName("alias")
	err := tf.Remove(hostType, alias)
	if err != nil {
		if err == prom.ErrHostNotFound {
			http.NotFound(w, r)
		} else {
			log.Println("ERROR: remove: tf.Remove:", err)
			proto.ErrorResponse(w, err)
		}
	} else {
		log.Printf("INFO: removed %s", alias)
		proto.JSONResponse(w, http.StatusOK, nil)
	}
}
