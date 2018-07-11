package main

import (
	"flag"
	"fmt"
	"github.com/laxmanvallandas/cloudconfig"
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
	var remoteUrl, remotePath, localPath, localFileName, localFileType, remoteFileType, configLocation string
	var dynamicCfg bool

	flag.StringVar(&remoteUrl, "cfgRemoteUrl", os.Getenv("CFG_REMOTE_URL"), "CFG_REMOTE_URL, Remote URL of etcd/Consul")
	flag.StringVar(&remotePath, "cfgRemotePath", os.Getenv("CFG_REMOTE_PATH"), "CFG_REMOTE_PATH, Key for ETCD or Consul")
	flag.StringVar(&localPath, "cfgLocalPath", os.Getenv("CFG_LOCAL_PATH"), "CFG_LOCAL_PATH, Path from which to read the config")
	flag.StringVar(&localFileName, "cfgLocalFileName", os.Getenv("CFG_LOCAL_FILENAME"), "CFG_LOCAL_FILENAME, Local FileName of Configuration without extension or filetype")
	flag.BoolVar(&dynamicCfg, "cfgDynamicConfig", true, "Enable or disable dynamic configuration by setting true or false")
	flag.StringVar(&localFileType, "cfgLocalFileType", os.Getenv("CFG_LOCAL_FILETYPE"), "CFG_LOCAL_FILETYPE, Local Filetype")
	flag.StringVar(&remoteFileType, "cfgremoteFileType", os.Getenv("CFG_REMOTE_FILETYPE"), "CFG_REMOTE_FILETYPE, Remote Filetype")
	flag.StringVar(&configLocation, "cfgConfigLocation", os.Getenv("CFG_CONFIG_LOCATION"), "CFG_CONFIG_LOCATION, Location of Configuration to pick from, local/remote. Empty means Local by default and tries Remote when local fails")

	flag.Parse()

	// Will go with below way to advertise viper config for now
	remoteCfgParams := cloudconfig.RemoteProvider{URL: remoteUrl, Path: remotePath}
	remoteConf := map[string]interface{}{"etcd": remoteCfgParams}
	viperConfig := map[string]interface{}{"localpath": localPath,
		"localpath_filename":  localFileName,
		"enabledynamicconfig": true,
		"remotepath":          remoteConf,
		"filetype":            localFileType,  //Local can be yml
		"remote_filetype":     remoteFileType} //Remote can be json

	app := Config{}
	cc, err := cloudconfig.InitCloudConfig(viperConfig, &app, configLocation)
	if err != nil {
		fmt.Println("Couldn't Initialise Cloud Config, Reason: ", err)
		os.Exit(1)
	}
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
