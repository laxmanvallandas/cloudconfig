cloud Configuration Wrapper library built on Viper library to support Local and Remote Configuration Loading and also for Dynamic Configuration.

Before using Wrapper, apply below viper.patch to Viper Library, also export getConfigFile api. by making GetConfigFile() or GetAppConfigFile() in viper as given below:

func (v *viper)GetConfigFile(string, error){
        return v.getConfigFile()
}

App should use "newconf" struct with possible dynamic config parameters. 

Library supports and test scenarios:
1: Load config file from local path.

2: Fail to load config file if config in local path is an invalid file format.(eg.invalid json). Revert back to working config file in local path.

3: Modify the local config file with new values in “newconf”, verify the callback invoked with new values in “newconf” struct in callback.

4: Modify the local config file with new values in “newconf” and make a invalid file format(eg. Invalid json/yml), 
        verify the callback invoked but throws an error and revert the configuration to old valid config.

5: Load config file from Remote Path served by etcd.

6: Fail to load config file if config in remote is an invalid file format (eg. Invalid json)

7: Modify the remote config file with new values in “newconf”, verify the callback invoked with new values in “newconf” struct in callback.

8: Modify the remote config file with new values in “newconf”, and make a invalid file format format(eg. Invalid json/yml), 
        verify the callback invoked but throws an error.

9: Disable dynamic configuration and any effect in local/remote config file should not invoke callback.


