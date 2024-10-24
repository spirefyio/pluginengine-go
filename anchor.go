package pluginengine

type Anchor struct {
	// a unique identifier such as a name made up of the company, org, project and category separated by periods, or just
	// a simple string. The point is that this is to be matched by hook's anchorId property in order for
	// the hook to resolve to the anchor when loaded.
	Id string `json:"id" yaml:"id"`

	// a meaningful description of this anchor that could be displayed in a plugin store for example. This
	// should probably provide details as to how the anchor will be called, when and what expectations if
	// any should be performed or provided by hooks
	Description string `json:"description" yaml:"description"`

	// a display or friendly name for this anchor, not to be confused with the Id.
	Name string `json:"name" yaml:"name"`

	// This is a schema an anchor would potentially send as a payload to a hook when the plugin code makes a call to
	// one (or more) hooks.
	CallSchema map[string]any `json:"callSchema" yaml:"callSchema"`

	// This is a schema expected in response from a hook back to the anchor plugin code. Hook implementations would need to 
	// return some subset of this structure in response to a call from an anchor.
	ResponseSchema map[string]any `json:"responseSchema" yaml:"responseSchema"`

	// This is an array of plugin Ids that must be available before this extension point can be used.
	Dependencies []Dependency `json:"dependencies" yaml:"dependencies"`
}

func CreateAnchor(id, name, version, description string, dependencies []Dependency) *Anchor {
	if len(id) > 0 && len(name) > 0 {
		return &Anchor{
			Id:           id,
			Description:  description,
			Name:         name,
			Dependencies: dependencies,
		}
	}

	return nil
}
