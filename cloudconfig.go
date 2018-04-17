package cloudconfig

import (
	"errors"
	"fmt"
	"time"
	"sync"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
)

//ConfigChangeCbk callback api to invoke application for config change events
type ConfigChangeCbk func(appConfig interface{}) bool

//CloudConfig - Wrapper config
type CloudConfig struct {
	AppConfig      interface{}
	viperConfig    map[string]interface{}
	Vi             *viper.Viper
	Info           appInfo
	cbk            ConfigChangeCbk
	configLocation string
	ConfigLock     *sync.RWMutex
}

type appInfo struct {
	EnableDynamicConfig bool
}

//RemoteProvider ...
type RemoteProvider struct {
	URL  string
	Path string
}
var (
	viDynamic *viper.Viper= viper.New()
	viLocal *viper.Viper = viper.New()
)
const (
	//LocalConfig -signifies Application preference to read config from local
	LocalConfig = "local"
	//RemoteConfig -signifies Application preference to read config from remote
	RemoteConfig = "remote"
	//Config - signifies Application preference to read either from local or remote , local being the first priority
	Config = ""
)

//InitCloudConfig - Api to be called from application
func InitCloudConfig(viperConfig map[string]interface{}, appConf interface{}, configLoc string) *CloudConfig {

	cc := &CloudConfig{Vi:viper.New(), AppConfig: appConf, configLocation: configLoc, ConfigLock: new(sync.RWMutex)}
	cc.populateViperConfig(viperConfig)
	err := cc.newConfigHandler(configLoc)
	if err != nil {
		fmt.Println("Could not Load Local config...", err, " Trying remote")
		return nil
	}
	return cc
}

func (c *CloudConfig) populateViperConfig(viperConfig map[string]interface{}) {
	fmt.Println("viper Config", viperConfig)
	for k, v := range viperConfig {
		switch k {
		case "localpath":
			fmt.Println("local path ", v.(string))
			viLocal.AddConfigPath(v.(string))
		case "localpath_filetype":
			viLocal.SetConfigType(v.(string))
		case "localpath_filename":
			fmt.Println(v.(string))
			viLocal.SetConfigName(v.(string))
		case "localpath_filewithtype":
			viLocal.SetConfigFile(v.(string))
		case "remotepath": //map[string]map[string]interface{} -> remotepath: etcd/consul: address
			for k2, v2 := range v.(map[string]interface{}) {
				remoteInfo := v2.(RemoteProvider)
				fmt.Println(remoteInfo)
				viDynamic.AddRemoteProvider(k2, remoteInfo.URL, remoteInfo.Path)
			}
		case "remote_filetype":
			viDynamic.SetConfigType(v.(string))
		case "remote_filename":
			fmt.Println(v.(string))
			viDynamic.SetConfigName(v.(string))
		case "remote_filewithtype":
			viDynamic.SetConfigFile(v.(string))
		case "enabledynamicconfig":
			fmt.Println("enable dynamic config ? ", v.(bool))
			c.Info.EnableDynamicConfig = v.(bool)
		}
	}
}

//RegisterConfigChange : Application Api to register for config change
func (c *CloudConfig) RegisterConfigChange(args ...interface{}) error {
	if len(args) != 1 {
		return errors.New("Help: Expected Args To Register for Config Change (<application Name>, args)")
	}
	c.cbk = args[0].(func(interface{}) bool) // callback to be implemented by app

	if c.configLocation == LocalConfig {
		c.Vi.WatchConfig()
		go c.monitorConfigChange()
	} else if c.configLocation == RemoteConfig {
		err := c.Vi.ReadRemoteConfig()
		if err != nil {
			returnErr("Could Not Read from Remote Config", err)
		}
		go c.monitorRemoteConfigChange()
	} else {
		c.Vi.WatchConfig()
		go c.monitorConfigChange() //this should go to thread

		go c.monitorRemoteConfigChange()
	}
	return nil
}

func (c *CloudConfig) newConfigHandler(configLoc string) error {
	var err error
	dynConf := false
	if c.configLocation == LocalConfig {
		if err = viLocal.ReadInConfig(); err != nil {
			return returnErr("Could Not Read Local Config ", err)
		}
		c.Vi = viLocal
	} else if c.configLocation == RemoteConfig {
		err := viDynamic.ReadRemoteConfig()
		if err != nil {
			return returnErr("Could Not Read from Remote Config", err)
		}
		c.Vi = viDynamic
	} else {
		if err = viLocal.ReadInConfig(); err != nil {
			err := viDynamic.ReadRemoteConfig()
			if err != nil {
				return returnErr("Could Not Read from Local/Remote Config", err)
			}
			dynConf = true
		}
		if dynConf {
			c.Vi = viDynamic
		}else{
			c.Vi = viLocal
		}
	}
	err = c.Vi.Unmarshal(&c.AppConfig)
	if err != nil {
		return returnErr("Failed to Unmarshal the remote config ", err)
	}
	return nil
}

func (c *CloudConfig) monitorRemoteConfigChange() {
	configChanged := make(chan bool)
	for {
		time.Sleep(time.Second * 5) // delay after each request

		// currently, only tested with etcd support
		err := c.Vi.WatchRemoteConfigOnChannel(configChanged)
		if err != nil {
			fmt.Println("unable to read remote config: ", err)
			continue
		}
		if !<-configChanged {
			continue
		}

		fmt.Println("Received config change event, unmarshalling new config")
		err = c.Vi.Unmarshal(&c.AppConfig)
		if err!= nil{
			fmt.Println("Error in Unmarshalling new Remote Config..")
			continue
		}

		//	fmt.Println(c.viDynamic.GetStringMap("newconf"))
		configChangedHandled := c.cbk(c.AppConfig)
		if !configChangedHandled {
			fmt.Println("Couldnt handle New config") //Reason can be returned from callback if required
			return
		}

	}
}

func (c *CloudConfig) monitorConfigChange() {
	c.Vi = viLocal
	for {
		time.Sleep(5 * time.Second)
		c.Vi.OnConfigChange(func(e fsnotify.Event) {
			if e.Op&fsnotify.Write == fsnotify.Write {
				err := c.newConfigHandler(LocalConfig)
				if err != nil {
					fmt.Println("found fault in new config , reloading old config ", err)
					c.reloadOrigConfig(c.AppConfig)
					return
				}
				configChangedHandled := c.cbk(c.AppConfig)
				if !configChangedHandled {
					fmt.Println("Couldnt handle New config") //Reason can be returned from callback if required
					return
				}
			}
		})
	}
}

func (c *CloudConfig) reloadOrigConfig(cfg interface{}) {
	file, err := c.Vi.GetAppConfigFile()
	if err != nil {
		fmt.Println("Could not get the old config file to reload")
		return
	}
	err = c.Vi.WriteConfigAs(file)
	if err != nil {
		fmt.Println("couldn't save the original config", err)
		return
	}

	c.newConfigHandler(LocalConfig)
}

//GetAppConfig ...
func (c *CloudConfig) GetAppConfig() interface{} {
	return c.AppConfig
}

func returnErr(str string, err error) error {
	fmt.Println("Error: ", str, "Reason:", err)
	return err
}
