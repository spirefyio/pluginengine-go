package pluginengine

import pdk "github.com/spirefyio/plugin-go-pdk"

type Plugin struct {
	// A unique id for this plugin. It may often contain the base id that anchors defined within the plugin
	// contain. For example mycompany.plugins.MyPlugin  and an anchor of this plugin might have an id of
	// mycompany.plugins.MyPlugin.MyAnchor
	Id string `json:"id" yaml:"id"`

	// A more readable name of the plugin
	Name string `json:"name" yaml:"name"`

	// The version of this plugin in SemVer format x.y.z
	Version string `json:"version" yaml:"version"`

	// A description of this plugin, what it does, provides, contributes to, etc.
	Description string `json:"description" yaml:"description"`

	// A slice of anchors that this plugin defines.. anchors for other plugins to extend from
	Anchors []pdk.Anchor `json:"anchors" yaml:"anchors"`

	// A slice of hooks that attach to other plugin anchors.. contributions this plugin is adding to those
	// anchors.
	Hooks []pdk.Hook `json:"hooks" yaml:"hooks"`

	LoadOnStart bool `json:"loadOnStart" yaml:"loadOnStart"`
}
