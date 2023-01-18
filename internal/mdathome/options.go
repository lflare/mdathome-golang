//go:build !linux
// +build !linux

package mdathome

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	bolt "go.etcd.io/bbolt"
)

func (c *Cache) getOptions() *bolt.Options {
	// Return no custom options because Windows does not support
	options := &bolt.Options{}
	return options
}

func prepareConfigurationReload() {
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Infof("Configuration updated: %v", viper.AllSettings())

		// Run manual configuration updates
		//// Update cache limits
		cache.UpdateCacheLimit(viper.GetInt(KeyCacheSize) * 1024 * 1024)
	})
	viper.WatchConfig()
}
