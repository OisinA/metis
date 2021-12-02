package cmd

import (
	"fmt"
	"metis/internal/api"
	"metis/pkg/config"
	"metis/pkg/provider"
	"net/http"

	"github.com/Strum355/log"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Starts a new metis agent",
	Run: func(cmd *cobra.Command, args []string) {
		startAgent()
	},
}

func startAgent() {
	config.Load()

	// Start the logger
	log.InitSimpleLogger(&log.Config{})
	log.Info("Starting Metis Client")

	config.PrintSettings()

	// Create the docker provider
	cli, err := provider.NewDockerProvider()
	if err != nil {
		panic(err)
	}

	log.Info("Docker provider started.")

	// Create the HTTP router
	r := chi.NewRouter()

	// Register the HTTP routes
	api := api.NewAPI(&cli)
	api.Register(r)

	log.Info("API registered.")

	log.WithFields(log.Fields{
		"port": viper.GetInt("metis.agent.port"),
	}).Info("Listening & serving")

	// Block on HTTP listening
	err = http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("metis.agent.port")), r)
	if err != nil {
		panic(err)
	}
}
