package main

import (
	"cloudconfig"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
)

// Config - Main config structure
type Config struct {
	AppName  string
	LogLevel string
	NewConf  DynamicConf
}

//DynamicConf - Add your dynamic config fields here
type DynamicConf struct {
	LogLevel string
}

var cc *cloudconfig.CloudConfig
var app2 Config

func main() {
	// Will go with below way to advertise viper config for now
	temp1 := cloudconfig.RemoteProvider{URL: "http://localhost:4001", Path: "/confignew2/test"}
	remoteConf := map[string]interface{}{"etcd": temp1}
	viperConfig := map[string]interface{}{"localpath": "/home/user/go_exercise/viper_sample/cfg",
		"localpath_filename":  "wsproxy",
		"enabledynamicconfig": true,
		"remotepath":          remoteConf,
		"filetype":            "yml",  //Local can be yml
		"remote_filetype":     "json"} //Remote can be json

	app := Config{}
	cc := cloudconfig.InitCloudConfig(viperConfig, &app, cloudconfig.LocalConfig)
	app2 = Config(app) // this should be configuration that app must use

	fmt.Println(app2)

	if cc.Info.EnableDynamicConfig {
		err := cc.RegisterConfigChange(dynamicConfigHandler)
		if err != nil {
			fmt.Println("error in registering ", err)
			os.Exit(1)
		}
	}
	http.HandleFunc("/getconfig", cc.GetCurrentRunningConf)
	log.Fatal(http.ListenAndServe(":8080", nil))
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	<-signals
}

func dynamicConfigHandler(appConf interface{}) bool {
	newconfig := appConf.(*Config)
	//Start playing with dynamic Configuration received as newconfig.NewConf ,

	// Remember to populate your main config,  app2
	fmt.Println(newconfig)

	return true
}
