// +build !wasm

package pluginengine

type Hook struct {
	// a unique identifier such as a name made up of the company, org, project and category separated by periods, or just
	// a simple string.
	Id string `json:"id" yaml:"id"`

	// the anchor Id that this hook provides functionality to/for.
	Anchor string `json:"anchorId" yaml:"anchorId"`

	// a meaningful description of this hook that could be displayed in a plugin store for example
	Description string `json:"description" yaml:"description"`

	// a display or friendly name for this hook, not to be confused with the Id.
	Name string `json:"name" yaml:"name"`

	// a func IN the WASM plugin that matches this value that would be called by the anchor using the plugin
	// engine's host function to call hook functions.
	Func string `json:"func" yaml:"func"`

	// This property can hold custom data for an anchor to use without having to call the hook func
	// to have data returned. This is useful when an anchor want's to build up say a Menu system or a Help
	// system that is composed of static data and does not require the hook func to execute to return such data.
	// Each anchor would define the format (structure) that this metadata would need to be in and it is up
	// to the exported register() function to marshall the data into a []byte before returning the hook(s) as
	// part of the register return []byte data. Anchor funcs would be called to process the metadata however
	// necessary.
	MetaData map[string]any `json:"metadata" yaml:"metadata"`

	// This is a slice of dependencies this hook depends on. If provided, the hook, anchro or
	// plugin in each dependency MUST resolved before the hook is usable (resolved)
	Dependencies []Dependency `json:"dependencies" yaml:"dependencies"`
}

func CreateHook(id, name, anchorId, description, funcName string, metadata map[string]any, dependencies []Dependency) *Hook {
	if len(id) > 0 && len(name) > 0 && len(anchorId) > 0 && len(funcName) > 0 {
		return &Hook{
			Id:           id,
			Anchor:       anchorId,
			Description:  description,
			Name:         name,
			Func:         funcName,
			MetaData:     metadata,
			Dependencies: dependencies,
		}
	}

	return nil
}
