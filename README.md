# cloudconfig
Config Library for Local and Remote Configuration. Support for Dynamic Configuration.


Before using Wrapper, apply below patch to Viper Library

diff --git a/viper.go b/viper.go
index e9966ba..27f72fb 100644
--- a/viper.go
+++ b/viper.go
@@ -1497,8 +1497,8 @@ func (v *Viper) WatchRemoteConfig() error {
        return v.watchKeyValueConfig()
 }
 
-func (v *Viper) WatchRemoteConfigOnChannel() error {
-       return v.watchKeyValueConfigOnChannel()
+func (v *Viper) WatchRemoteConfigOnChannel(configChanged chan<- bool) error {
+       return v.watchKeyValueConfigOnChannel(configChanged)
 }
 
 func (v *Viper) insensitiviseMaps() {
@@ -1535,7 +1535,7 @@ func (v *Viper) getRemoteConfig(provider RemoteProvider) (map[string]interface{}
 }
 
 // Retrieve the first found remote configuration.
-func (v *Viper) watchKeyValueConfigOnChannel() error {
+func (v *Viper) watchKeyValueConfigOnChannel(configChanged chan<- bool) error {
        for _, rp := range v.remoteProviders {
                respc, _ := RemoteConfig.WatchChannel(rp)
                //Todo: Add quit channel
@@ -1544,6 +1544,7 @@ func (v *Viper) watchKeyValueConfigOnChannel() error {
                                b := <-rc
                                reader := bytes.NewReader(b.Value)
                                v.unmarshalReader(reader, v.kvstore)
+                               configChanged <- true
                        }
                }(respc)
                return nil

