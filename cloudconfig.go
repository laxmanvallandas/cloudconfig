package cloudconfig

import (
	"errors"
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
)

//ConfigChangeCbk ...
type ConfigChangeCbk func(appConfig interface{}) bool

//CloudConfig - Wrapper config
type CloudConfig struct {
	AppConfig      interface{}
	viperConfig    map[string]interface{}
	vi             *viper.Viper
	viDynamic      *viper.Viper
	Info           appInfo
	cbk            ConfigChangeCbk
	configLocation string
}

type appInfo struct {
	EnableDynamicConfig bool
}

//RemoteProvider ...
type RemoteProvider struct {
	URL  string
	Path string
}

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

	cc := &CloudConfig{vi: viper.New(), AppConfig: appConf, viDynamic: viper.New(), configLocation: configLoc}
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
			c.vi.AddConfigPath(v.(string))
		case "localpath_filetype":
			c.vi.SetConfigType(v.(string))
		case "localpath_filename":
			fmt.Println(v.(string))
			c.vi.SetConfigName(v.(string))
		case "localpath_filewithtype":
			c.vi.SetConfigFile(v.(string))
		case "remotepath": //map[string]map[string]interface{} -> remotepath: etcd/consul: address
			fmt.Println("remotepath", v)
			for k2, v2 := range v.(map[string]interface{}) {
				remoteInfo := v2.(RemoteProvider)
				fmt.Println(remoteInfo)
				c.viDynamic.AddRemoteProvider(k2, remoteInfo.URL, remoteInfo.Path)
			}
		case "remote_filetype":
			c.viDynamic.SetConfigType(v.(string))
		case "remote_filename":
			fmt.Println(v.(string))
			c.viDynamic.SetConfigName(v.(string))
		case "remote_filewithtype":
			c.viDynamic.SetConfigFile(v.(string))
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
		c.vi.WatchConfig()
		go c.monitorConfigChange()
	} else if c.configLocation == RemoteConfig {
		err := c.viDynamic.ReadRemoteConfig()
		if err != nil {
			return returnErr("Could Not Read from Remote Config", err)
		}
		go c.monitorRemoteConfigChange()
	} else {
		c.vi.WatchConfig()
		go c.monitorConfigChange() //this should go to thread

		go c.monitorRemoteConfigChange()
	}
	return nil
}

func (c *CloudConfig) newConfigHandler(configLoc string) error {
	var viTemp *viper.Viper
	var err error
	dynConf := false
	if c.configLocation == LocalConfig {
		if err = c.vi.ReadInConfig(); err != nil {
			return returnErr("Could Not Read Local Config ", err)
		}
		viTemp = c.vi
	} else if c.configLocation == RemoteConfig {
		err := c.viDynamic.ReadRemoteConfig()
		if err != nil {
			return returnErr("Could Not Read from Remote Config", err)
		}
		viTemp = c.viDynamic
	} else {
		if err = c.vi.ReadInConfig(); err != nil {
			err := c.viDynamic.ReadRemoteConfig()
			if err != nil {
				return returnErr("Could Not Read from Local/Remote Config", err)
			}
			viTemp = c.viDynamic
			dynConf = true
		}
		if dynConf {
			viTemp = c.viDynamic
		}
		viTemp = c.vi
	}
	err = viTemp.Unmarshal(&c.AppConfig)
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
		err := c.viDynamic.WatchRemoteConfigOnChannel(configChanged)
		if err != nil {
			fmt.Println("unable to read remote config: ", err)
			continue
		}

		if !<-configChanged {
			continue
		}

		c.viDynamic.Unmarshal(&c.AppConfig)

		//	fmt.Println(c.viDynamic.GetStringMap("newconf"))
		configChangedHandled := c.cbk(c.AppConfig)
		if !configChangedHandled {
			fmt.Println("Couldnt handle New config") //Reason can be returned from callback if required
			return
		}

	}
}

func (c *CloudConfig) monitorConfigChange() {
	for {
		time.Sleep(5 * time.Second)
		c.vi.OnConfigChange(func(e fsnotify.Event) {
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
	file, err := c.vi.GetConfigFile()
	if err != nil {
		fmt.Println("Could not get the old config file to reload")
		return
	}
	err = c.vi.WriteConfigAs(file)
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
