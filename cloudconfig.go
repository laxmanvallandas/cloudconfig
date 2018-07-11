package cloudconfig

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/laxmanvallandas/viper"
	_ "github.com/laxmanvallandas/viper/remote"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

//ConfigChangeCbk ...
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
	viRemote *viper.Viper = viper.New()
	viLocal  *viper.Viper = viper.New()
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
func InitCloudConfig(viperConfig map[string]interface{}, appConf interface{}, configLoc string) (*CloudConfig, error) {

	cc := &CloudConfig{AppConfig: appConf, configLocation: configLoc, ConfigLock: new(sync.RWMutex)}
	cc.populateViperConfig(viperConfig)
	err := cc.newConfigHandler(configLoc)
	if err != nil {
		fmt.Println("Could not Load Local config...", err, " Trying remote")
		return nil, err
	}
	return cc, nil
}

func (c *CloudConfig) populateViperConfig(viperConfig map[string]interface{}) {
	for k, v := range viperConfig {
		switch k {
		case "localpath":
			viLocal.AddConfigPath(v.(string))
		case "localpath_filetype":
			viLocal.SetConfigType(v.(string))
		case "localpath_filename":
			viLocal.SetConfigName(v.(string))
		case "localpath_filewithtype":
			viLocal.SetConfigFile(v.(string))
		case "remotepath": //map[string]map[string]interface{} -> remotepath: etcd/consul: address
			for k2, v2 := range v.(map[string]interface{}) {
				remoteInfo := v2.(RemoteProvider)
				viRemote.AddRemoteProvider(k2, remoteInfo.URL, remoteInfo.Path)
			}
		case "remote_filetype":
			viRemote.SetConfigType(v.(string))
		case "remote_filename":
			viRemote.SetConfigName(v.(string))
		case "remote_filewithtype":
			viRemote.SetConfigFile(v.(string))
		case "enabledynamicconfig":
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
		viLocal.WatchConfig()
		go c.monitorConfigChange() //this should go to thread

		go c.monitorRemoteConfigChange()
	}
	return nil
}

func (c *CloudConfig) newConfigHandler(configLoc string) error {
	var err error
	remoteConf := false
	if c.configLocation == LocalConfig {
		if err = viLocal.ReadInConfig(); err != nil {
			return returnErr("Could Not Read Local Config ", err)
		}
		c.Vi = viLocal
	} else if c.configLocation == RemoteConfig {
		err := viRemote.ReadRemoteConfig()
		if err != nil {
			return returnErr("Could Not Read from Remote Config", err)
		}
		c.Vi = viRemote
	} else {
		if err = viLocal.ReadInConfig(); err != nil {
			err := viRemote.ReadRemoteConfig()
			if err != nil {
				return returnErr("Could Not Read from Local/Remote Config", err)
			}
			remoteConf = true
		}
		if remoteConf {
			c.Vi = viRemote
		} else {
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
	err := viRemote.ReadRemoteConfig()
	if err != nil {
		return
	}

	configChanged := make(chan bool)
	for {
		time.Sleep(time.Second * 5) // delay after each request

		// currently, only tested with etcd support
		err := viRemote.WatchRemoteConfigOnChannel(configChanged)
		if err != nil {
			continue
		}
		if !<-configChanged {
			continue
		}

		err = viRemote.Unmarshal(&c.AppConfig)
		if err != nil {
			continue
		}

		//	fmt.Println(c.viRemote.GetStringMap("newconf"))
		configChangedHandled := c.cbk(c.AppConfig)
		if !configChangedHandled {
			return
		}
		c.Vi = viRemote

	}
}

func (c *CloudConfig) monitorConfigChange() {
	for {
		time.Sleep(5 * time.Second)
		c.Vi.OnConfigChange(func(e fsnotify.Event) {
			if e.Op&fsnotify.Write == fsnotify.Write {
				err := c.newConfigHandler(LocalConfig)
				if err != nil {
					c.reloadOrigConfig(c.AppConfig)
					return
				}
				configChangedHandled := c.cbk(c.AppConfig)
				if !configChangedHandled {
					return
				}
				c.Vi = viLocal
			}
		})
	}
}

func (c *CloudConfig) reloadOrigConfig(cfg interface{}) {
	file, err := c.Vi.GetAppConfigFile()
	if err != nil {
		return
	}
	err = c.Vi.WriteConfigAs(file)
	if err != nil {
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

func (cc *CloudConfig) GetCurrentRunningConf(w http.ResponseWriter, r *http.Request) {
	f, err := os.Create("/tmp/currentconfig.json")
	if err != nil {
		fmt.Println("could not create file")
		return
	}
	fmt.Println("cc is ", cc)
	err = cc.Vi.WriteConfigAs("/tmp/currentconfig.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	defer f.Close()

	file, err := os.Open("/tmp/currentconfig.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	n, err := io.Copy(w, file)
	if err != nil {
		fmt.Println("error in copying from file response writer ", err)
		return
	}
	fmt.Println(n)
}

func (cc *CloudConfig) GetConfig(configKey string, w http.ResponseWriter) {
	var vi *viper.Viper
	fmt.Println("config key ", configKey)
	if cc.Vi.IsSet(configKey) {
		vi = cc.Vi.Sub(configKey)
		vi.SetConfigType(cc.Vi.GetConfigType())
		err := getConfig(w, vi)
		if err != nil {
			fmt.Println("Could not Get requested Config: ", err)
			return
		}
	} else if configKey == "" {
		err := getConfig(w, cc.Vi)
		if err != nil {
			fmt.Println("Could not Get requested Config: ", err)
			return
		}
	} else {
		http.Error(w, "Config Not Found", 400)
	}
}

func getConfig(w http.ResponseWriter, v *viper.Viper) error {
	f, err := os.Create("/tmp/config." + v.GetConfigType())
	if err != nil {
		http.Error(w, "Could not Get Current Config file : "+err.Error(), 500)
		return err
	}
	defer f.Close()

	err = v.WriteConfigAs("/tmp/config." + v.GetConfigType())
	if err != nil {
		http.Error(w, "Could not Get Current Config file : "+err.Error(), 500)
		return err
	}

	file, err := os.Open("/tmp/config." + v.GetConfigType())
	if err != nil {
		http.Error(w, "Could not Get Current Config file : "+err.Error(), 500)
		return err
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Could not Get Current Config file : "+err.Error(), 500)
		return err
	}
	return nil
}
