package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// KubexyArgs stores passed arguments
type KubexyArgs struct {
	KubeConfig     string
	AnonymizeData  bool
	DemoMode       bool
	DemoNodes      int
	DemoNamespaces int
	DemoPods       int
	DemoSpeed      int
}

// KubexyApp has runtime state and variables
type KubexyApp struct {
	Args  KubexyArgs
	Views map[string]View
}

// View is a base interface to collect data for view
type View interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// ConfigureServer returns configured http server
func (app *KubexyApp) ConfigureServer() *http.Server {

	r := mux.NewRouter()
	for view, impl := range app.Views {
		r.Handle("/view/"+view, impl)
	}
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("html/")))

	return &http.Server{
		Handler:      handlers.LoggingHandler(os.Stdout, r),
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}

func main() {

	app := KubexyApp{}
	args := &app.Args
	flag.StringVar(&args.KubeConfig, "kubeconfig", "", "path to kubeconfig file with in-cluster detection")
	flag.BoolVar(&args.AnonymizeData, "anonymize", false, "anonymize your cluster data")
	flag.BoolVar(&args.DemoMode, "demo", false, "simulate working cluster")
	flag.IntVar(&args.DemoNodes, "demo-nodes", 6, "demo-mode: number of nodes")
	flag.IntVar(&args.DemoNamespaces, "demo-namespaces", 5, "demo-mode: number of namespaces")
	flag.IntVar(&args.DemoPods, "demo-pods", 15, "demo-mode: number of pods")
	flag.IntVar(&args.DemoSpeed, "demo-speed", 500, "demo-mode: simulation speed 0 - 100 (0-slow, 100-fast)")
	flag.Parse()

	log.Printf("KubeXY started, configuration %+v", app.Args)

	rand.Seed(time.Now().UTC().UnixNano())

	// create views
	app.Views = make(map[string]View)
	if args.DemoMode {
		app.Views["pods"] = NewViewPodDemo(app)
	} else {
		app.Views["pods"] = NewViewPodKube(app)
	}

	// start server
	server := app.ConfigureServer()
	log.Fatal(server.ListenAndServe())
}
