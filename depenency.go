// +build !wasm

package pluginengine

type Dependency struct {
	PluginId   string `json:"pluginId" yaml:"pluginId"`
	MinVersion string `json:"minVersion" yaml:"minVersion"`
	MaxVersion string `json:"maxVersion" yaml:"maxVersion"`
}
