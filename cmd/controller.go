package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"metis/pkg/config"
	"metis/pkg/node"
	"metis/pkg/orchestrator"
	"metis/pkg/project"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Strum355/log"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"
)

const (
	VERSION = "0.1.0"
)

var controllerCmd = &cobra.Command{
	Use:   "controller",
	Short: "Start a new metis controller",
	Run: func(cmd *cobra.Command, args []string) {
		startController()
	},
}

func startController() {
	config.Load()

	log.InitSimpleLogger(&log.Config{})
	log.Info("Starting Metis Client")

	config.PrintSettings()

	orchestrator := orchestrator.NewOrchestrator()

	node_files, err := ioutil.ReadDir("nodes")
	if err != nil {
		panic(err)
	}

	nodes := make(map[string]node.Node)
	for i, f := range node_files {
		byts, err := ioutil.ReadFile("nodes/" + f.Name())
		if err != nil {
			panic(err)
		}
		node := node.Node{ID: fmt.Sprintf("node-%d", i)}
		err = json.Unmarshal(byts, &node)
		if err != nil {
			panic(err)
		}
		nodes[node.ID] = node
	}

	orchestrator.Nodes = nodes

	orchestrator.NodeHealthcheck()
	fmt.Println(orchestrator.Nodes)

	files, err := ioutil.ReadDir("projects")
	if err != nil {
		panic(err)
	}

	projects := make([]project.Project, 0)
	for _, f := range files {
		byts, err := ioutil.ReadFile("projects/" + f.Name())
		if err != nil {
			panic(err)
		}
		proj := project.Project{}
		err = json.Unmarshal(byts, &proj)
		if err != nil {
			panic(err)
		}
		projects = append(projects, proj)
	}

	for _, proj := range projects {
		err = orchestrator.CreateProject(proj)

		if err != nil {
			panic(err)
		}
	}

	go func() {
		r := chi.NewRouter()
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			err := json.NewEncoder(w).Encode(struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}{
				"METIS", VERSION,
			})
			if err != nil {
				log.WithError(err).Error("Could not send API response")
				return
			}
		})

		r.Get("/projects", func(w http.ResponseWriter, r *http.Request) {
			projects := []struct {
				Healthy int `json:"healthy"`
				project.Project
			}{}
			for _, proj := range orchestrator.Projects {
				healthy, err := orchestrator.CountHealthy(proj.Name)
				if err != nil {
					log.WithError(err).Error("Could not return project")
					continue
				}
				response := struct {
					Healthy int `json:"healthy"`
					project.Project
				}{
					Healthy: healthy,
					Project: proj,
				}
				projects = append(projects, response)
			}
			err := json.NewEncoder(w).Encode(projects)
			if err != nil {
				log.WithError(err).Error("Could not send API response")
				return
			}
		})

		r.Get("/services", func(w http.ResponseWriter, r *http.Request) {
			err := json.NewEncoder(w).Encode(orchestrator.GetServices())
			if err != nil {
				log.WithError(err).Error("Could not send API response")
				return
			}
		})

		r.Get("/nodes", func(w http.ResponseWriter, r *http.Request) {
			err := json.NewEncoder(w).Encode(orchestrator.Nodes)
			if err != nil {
				log.WithError(err).Error("Could not send API response")
				return
			}
		})

		r.Get("/traefik", func(w http.ResponseWriter, r *http.Request) {
			config := orchestrator.GetTraefikConfig()

			err := json.NewEncoder(w).Encode(config.ToMap())
			if err != nil {
				w.WriteHeader(500)
				log.WithError(err).Error("Could not create traefik config")
			}
		})

		log.Info("Started API service")

		err := http.ListenAndServe(":8060", r)
		if err != nil {
			panic(err)
		}
	}()

	shutdownCh := make(chan struct{})

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		close(shutdownCh)
	}()

	for {
		select {
		case <-shutdownCh:
			return
		default:
		}
		err = orchestrator.Update()
		if err != nil {
			log.WithError(err).Error("Error updating orchestrator")
		}
		time.Sleep(5 * time.Second)
	}

}
