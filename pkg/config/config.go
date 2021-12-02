package config

import (
	"encoding/json"
	"strings"

	"github.com/Strum355/log"
	"github.com/spf13/viper"
)

func Load() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	loadDefaults()
	viper.AutomaticEnv()
}

func loadDefaults() {
	viper.SetDefault("metis.home", "metis/data")
	viper.SetDefault("metis.agent.port", "6060")
	viper.SetDefault("metis.secret", "1oldmsmkp!")
	viper.SetDefault("metis.controller.url", "localhost")
}

func PrintSettings() {
	settings := viper.AllSettings()

	out, _ := json.MarshalIndent(settings, "", "\t")
	log.Debug("config:\n" + string(out))
}
