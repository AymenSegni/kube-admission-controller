package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/spf13/viper"
	"repos.ambidexter.gmbh/devops/admission-controller/admission"
)

func main() {
	// Load json config file
	viper.SetConfigName("local")
	viper.AddConfigPath("./config/")
	viper.SetConfigType("json")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}

	// ENV overwrites
	viper.SetEnvPrefix("AMBIDEXTER")
	viper.AutomaticEnv()

	viper.SetDefault("debug", true)
	debug := viper.GetBool("debug")
	viper.SetDefault("loglevel", "")
	logLevel := viper.GetString("loglevel")
	viper.SetDefault("denyLatestTag", true)
	denyLatestTag := viper.GetBool("denyLatestTag")
	registryWhitelist := viper.GetStringSlice("registryWhiteList")

	viper.SetDefault("port", 443)
	port := viper.GetInt("port")
	viper.SetDefault("servercert", "")
	serverCert := viper.GetString("servercert")
	viper.SetDefault("serverkey", "")
	serverKey := viper.GetString("serverkey")

	if len(registryWhitelist) > 0 {
		log.Infof("Accepting only images from registries: %+v", registryWhitelist)
	} else {
		log.Warn("Accepting images from ALL registries")
	}

	e := echo.New()
	e.POST("/pods", admission.AdmitPods(denyLatestTag, registryWhitelist))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339_nano}","remote_ip":"${remote_ip}","host":"${host}",` +
			`"method":"${method}","uri":"${uri}","status":${status},"latency_micro_sec":"${latency}"}` + "\n",
		Output: os.Stdout}))

	if debug {
		log.Info("Setting debug level logging")
		e.Logger.SetLevel(log.DEBUG)
	} else if strings.ToLower(logLevel) == "info" {
		log.Info("Setting info level logging")
		e.Logger.SetLevel(log.INFO)
	}

	if serverCert != "" && serverKey != "" {
		log.Infof("Starting HTTPS server with certs [%s %s]", serverCert, serverKey)
		e.StartTLS(
			fmt.Sprintf(":%d", port),
			serverCert,
			serverKey)
	} else {
		panic("Kubernetes api-server can only talk to a TLS enabled web server")
	}

}
