cloudconfig For MicroServices
=============================

BACKGROUND
-------------
cloudconfig is a Cloud Configuration Wrapper library built on Viper library to support Local and Remote Configuration Loading and also for Dynamic Configuration.
Library also provides Rest Interface using which client can get current runnig configuration.


PREREQUISITES
-------------

- This requires Go 1.5 or later
- Requires that [GOPATH is set](https://golang.org/doc/code.html#GOPATH)
- App should use "newconf" struct with possible dynamic config parameters.
```
$ go help gopath
$ # ensure the PATH contains $GOPATH/bin
$ export PATH=$PATH:$GOPATH/bin
```

INSTALL
-------

```
$ go get github.com/laxmanvallandas/cloudconfig
$ cd $GOPATH/src/github.com/laxmanvallandas/cloudconfig/examples

```

TRY IT!
-------

- Run the sample application

```
$ go build cloudconfig_example.go
$ Look for the env's to set ./cloudconfig_example --help
$ ./cloudconfig_example
```

- Get the configuration using http://<ip>:8080/getconfig
- Modify the configuration Local/Remote (based on your env settings) and you should see event generated to application.

```

Serves Following Purpose:
----------------------------------------
1 Load config file from local path.

2 Fail to load config file if config in local path is an invalid file format.(eg.invalid json). Revert back to working config file in local path.

3 Modify the local config file with new values in “newconf”, verify the callback invoked with new values in “newconf” struct in callback.

4 Modify the local config file with new values in “newconf” and make a invalid file format(eg. Invalid json/yml),  verify the callback invoked but throws an error and revert the configuration to old valid config.

5 Load config file from Remote Path served by etcd.

6 Fail to load config file if config in remote is an invalid file format (eg. Invalid json)

7 Modify the remote config file with new values in “newconf”, verify the callback invoked with new values in “newconf” struct in callback.

8 Modify the remote config file with new values in “newconf”, and make a invalid file format format(eg. Invalid json/yml), 
        verify the callback invoked but throws an error.

9 Disable dynamic configuration and any effect in local/remote config file should not invoke callback.


```
