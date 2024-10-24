package pluginengine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	extism "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"
	"gopkg.in/yaml.v3"
)

type (
	anchor struct {
		Anchor `json:"anchor" yaml:"anchor"`
		// Because this outer ExtensionPoint wrapper allows for host extension points, which are native to Go, a func pointer
		// to call upon that extension point is necessary. This is not the typical wasm string func name to call, but an
		// actual Go function provided by the host to be called
		Func   func([]*hook) error
		Hooks  []*hook `json:"hooks" yaml:"hooks"`
		Plugin plugin  `json:"plugin" yaml:"plugin"`
	}

	hook struct {
		Hook     `json:"hook" yaml:"hook"`
		Plugin   plugin `json:"plugin" yaml:"plugin"`
		Resolved bool   `json:"resolved" yaml:"resolved"`
	}

	plugin struct {
		Plugin       *extism.Plugin `json:"plugin" yaml:"plugin"`
		PathToModule string         `json:"pathToModule" yaml:"pathToModule"`
		Resolved     bool           `json:"resolved" yaml:"resolved"`
		LoadOnStart  bool           `json:"loadOnStart" yaml:"loadOnStart"`
	}

	Engine struct {
		context    context.Context
		logLevel   extism.LogLevel
		plugins    map[string]map[string]*plugin
		anchors    map[string][]*anchor
		hooks      map[string]*hook
		unresolved []*hook
		hostFuncs  []extism.HostFunction
		pluginPath string // path where .tar.gz and .zip plugins will be extracted to (overwrite every time)
	}
)

// This variable will hold pointers to plugins keyed on an hook ID. This can be used
// by CallHook to quickly access the hook to make a call to and check if its instantiated
// or not and resolved.
var callableHooks = make(map[string]*plugin)

func findFilesWithExtensions(root string, extensions []string) ([]string, error) {
	var matchingFiles []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		for _, ext := range extensions {
			if !entry.IsDir() && strings.HasSuffix(path, ext) {
				// You can also read the file contents here if needed
				matchingFiles = append(matchingFiles, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matchingFiles, nil
}

// addPlugin
//
// This method will add the plugin passed in to the engine's plugins property. It will ensure that if a plugin
// at the name and version provided does not yet exist, the map of internalPlugin objects is created.
// It's important to note that if a plugin already exists at the name and version intersection, it is replaced. This
// should allow for reloading (and eventual GC of old plugins as they are replaced) if need be.
func (e *Engine) addPlugin(p *plugin, plug Plugin) {
	if nil != e.plugins && nil != p {
		pv := e.plugins[plug.Id]

		if nil == pv {
			pv = make(map[string]*plugin)
			e.plugins[plug.Id] = pv
		}

		pv[plug.Version] = p
		p.LoadOnStart = plug.LoadOnStart

		// now add all of this plugins hooks to the unresolved list... a call to engine.resolve() will then try to
		// find/resolve all extensions and subsequently resolve all plugins
		if nil != plug.Hooks && len(plug.Hooks) > 0 {
			for _, ex := range plug.Hooks {
				hk := &hook{
					Hook:     ex,
					Plugin:   *p,
					Resolved: false,
				}

				// for each extension, add a reference pointer to THIS plugin so that when calling any extension
				// that is part of the same plugin owner, the pointer to the extism.Plugin instance can be used.
				if callableHooks[ex.Id] != nil {
					fmt.Println("It appears an extension is already added to the callableExtensions at id: ", ex.Id)
				} else {
					callableHooks[ex.Id] = p
				}

				e.unresolved = append(e.unresolved, hk)
			}
		}

		// now add all the plugins extension points to the engines extension points using the ExtensionPoint object
		// that will tie this plugin instance to it as well.

		if nil != plug.Anchors && len(plug.Anchors) > 0 {
			for _, ep := range plug.Anchors {
				eep := &anchor{
					Anchor: ep,
					Func:   nil,
					Hooks:  nil,
					Plugin: *p,
				}

				eps := e.anchors[ep.Id]
				if nil == eps {
					eps = make([]*anchor, 0)
				}

				eps = append(eps, eep)
				// reassign because exps may be a new larger ref.. has to be reassigned
				e.anchors[ep.Id] = eps
			}
		}
	}

	e.resolve()
}

// validateVersion
//
// TODO: For now this just returns true. It needs to add a check to make sure a version string matches a semver version
// value
func isSemverValid(version string) bool {
	// Split the version string into major, minor, and patch components
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	// Check if each component is a non-negative integer
	for _, part := range parts {
		if !isValidNumber(part) {
			return false
		}
	}
	return true
}

// isValidNumber
// helper func used by isSemverValid
func isValidNumber(str string) bool {
	if len(str) == 0 || str[0] == '-' {
		return false
	}

	for _, c := range str {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// GetExtensionForId
//
// This function will look for a single extension based on it's id (and version?) and return it if found, nil otherwise
func (e *Engine) GetHookForId(eid string) *Hook {
	hk := e.hooks[eid]

	if nil != hk && hk.Resolved {
		return &hk.Hook
	}

	return nil
}

// GetHooksForAnchor
//
// This method will look for a matching hook in the map of hooks and if found and the version provided is not
// nil, look for a matching version (TODO: version range may be added in future). If version is nil, the first
// anchr's hooks are returned.
func (e *Engine) GetHooksForAnchor(anchorId string) ([]*Hook, error) {
	fmt.Println("Looking for anchor: ", anchorId)
	anchrs := e.anchors[anchorId]

	if nil != anchrs && len(anchrs) > 0 {
		for _, anchrVer := range anchrs {
			hks := make([]*Hook, 0)
			for _, ahk := range anchrVer.Hooks {
				hks = append(hks, &ahk.Hook)
			}
		}
	}

	return nil, errors.New("no extensions found for extension point")
}

// getPluginName
// the source will be a .zip or .tar.gz so we'll remove those first, then look for the first / and get the name from
// everything past the / to the end
func getPluginName(source string) string {
	var name string
	if strings.HasSuffix(source, ".tar.gz") {
		name = source[:len(source)-7]
	} else if strings.HasSuffix(source, ".zip") {
		name = source[:len(source)-4]
	} else {
		// TODO: Log that the source file is NOT a .zip or .tar.gz
		return ""
	}

	// make sure we got a valid string still after removal of suffix
	if len(name) > 0 {
		indx := strings.LastIndex(name, string(filepath.Separator))
		if indx >= 0 {
			name = name[indx+1:]
			return name
		}
	}

	return ""
}

// loadPluginManifests
//
// This receiver function will be called to find all .tar.gz and .zip plugins at the provided path. It will determine if
// it's a .tar.gz or .zip and use the appropriate helper func to untar/unzip to the engine's pluginPath output location
// on the local file system. This extraction is necessary so that the .yaml (or .json (tbd)) can be parsed to pull the
// plugin details, as well as record the location of the .wasm plugin for later use when the plugin is instantiated.
func (e *Engine) loadPluginManifests(path, ext string) error {
	// Hardcode WASM extension as it's the only plugin module format supported.
	files, err := findFilesWithExtensions(path, []string{".gz", ".zip"})

	if err != nil {
		// Handle error
		fmt.Println("Some sort of error looking for .tar.gz or .zip plugin archive files: ", err)
		return err
	}

	// we need to extract the plugin archives to the plugin engine provided output path
	for _, file := range files {
		f := getPluginName(file)

		outputPath := filepath.Join(e.pluginPath, f)

		if strings.HasSuffix(file, ".tar.gz") {
			err = Untar(file, outputPath)
			if err != nil {
				// TODO: Log error.. but do NOT return because other plugins can still be extracted/loaded and work fine
				fmt.Println("Error unzipping .tar.gz plugin: ", file, err)
			}
		} else if strings.HasSuffix(file, ".zip") {
			err = Unzip(file, outputPath)
			if err != nil {
				// TODO: Log error.. but do NOT return because other plugins can still be extracted/loaded and work fine
				fmt.Println("Error unzipping zip plugin: ", file, err)
			}
		} else if strings.HasSuffix(file, ext) {
			err = Unzip(file, outputPath)
			if err != nil {
				// TODO: Log error.. but do NOT return because other plugins can still be extracted/loaded and work fine
				fmt.Println("Error unzipping zip plugin: ", file, err)
			}
		}

		// Because an error could occur, but we're in a loop that needs to process potentially multiple plugins, we're
		// checking if the error is nil
		if nil == err {
			// looking for the extracted yaml plugin descriptor manifest file
			files, err := findFilesWithExtensions(outputPath, []string{".yaml"})
			if nil != err {
				fmt.Println("Error trying to find YAML plugin manifest files")
			} else if len(files) > 0 {
				for _, f := range files {
					// grab the base path where the plugin was extracted
					base, _ := filepath.Split(f)

					// get the WASM file
					wasm, err2 := findFilesWithExtensions(base, []string{".wasm"})
					if nil != err2 {
						fmt.Println("Error walking base path: ", err2)
					}

					// read the bytes of the configuration file in
					data, err := os.ReadFile(f)
					if err != nil {
						fmt.Println("Error reading file:", err)
					}

					p := Plugin{}
					err = yaml.Unmarshal(data, &p)

					if nil != err {
						fmt.Println("Got error unmarshalling: ", err)
					} else {
						plug := &plugin{
							PathToModule: wasm[0],
							Plugin:       nil,
							Resolved:     false,
						}

						// register plugin, extension points and extensions
						e.addPlugin(plug, p)
					}
				}
			}
		}
	}

	return nil
}

// instantiate
//
// this function will create the plugin instance and call the plugin's start lifecycle exported function. This
// function should be called when another plugin's extension function is to be called and the plugin is not yet created
func (e *Engine) instantiate(plugin *plugin) error {
	ctx := e.context
	compilationCache := wazero.NewCompilationCache()
	defer func(cache wazero.CompilationCache, ctx context.Context) {
		err := cache.Close(ctx)
		if err != nil {
			fmt.Println("Error closing cache: ", err)
		}
	}(compilationCache, ctx)

	config := extism.PluginConfig{
		EnableWasi:    true,
		ModuleConfig:  wazero.NewModuleConfig(),
		RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(compilationCache),
	}

	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{
				Path: plugin.PathToModule,
			},
		},
	}

	pluginInstance, err := extism.NewPlugin(ctx, manifest, config, e.hostFuncs)

	if err != nil {
		fmt.Printf("Failed to initialize plugin: %v\n", err)
	}

	plugin.Plugin = pluginInstance

	_, _, err = pluginInstance.Call("start", nil)

	if nil != err {
		fmt.Println("Error calling plugin: ", err)
	}
	//} else {
	//	return errors.New("can not instantiate a plugin that is not yet resolved: " + plugin.Details.Id)
	// }

	return nil
}

// Start
//
// This method is called by an application to start the engine. This should occur after the Load() has finished and all
// plugins are found/parsed/resolved. Start will cycle through all plugins to find any with a startOnLoad flag which
// would indicate the plugin should be instantiated. For plugins that do not have startOnLoad set, they will be
// instantiated when first used via a call to an extension.
func (e *Engine) Start() error {
	for _, plugin := range e.plugins {
		if len(plugin) > 0 {
			for _, verPlugin := range plugin {
				if verPlugin.LoadOnStart {
					err := e.instantiate(verPlugin)

					if nil != err {
						fmt.Println("Error instantiating plugin: ", err)
					}
				}
			}
		}
	}

	return nil
}

// Load
//
// This recv/func is going to load plugins found in the provided path on the local filesystem. This path should be an
// absolute path on a local file system or a URL to an archived plugin file. The archive needs to be in a .tar.gz or
// .zip format. If the path provided is an http/https location, it will download the plugin to the engine plugin path
// and then unzip/untar it there.
func (e *Engine) Load(path string) error {
	// First make sure that path is NOT a URL to a single plugin file
	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "http") {
		// This is a URL
		u, err := url.Parse(lower)
		fmt.Println("u, err: ", u, err)
		// return for now as nil since we're not doing URLs yet
		// TODO: FIX THIS
		return nil
	}

	base, err := os.Getwd()
	if err != nil {
		return err
	}

	newPath := filepath.Join(base, lower)

	err = e.loadPluginManifests(newPath, "")
	if nil != err {
		fmt.Println("Error loading plugins: ", err)
	}

	e.resolve()
	return nil
}

// resolve
//
// This method will loop through all plugins and unresolved extensions, attempting to ensure all extensions of a given
// plugin have been resolved to loaded plugins with matching extension points. Only when all extensions of a plugin
// are resolved will a plugin's status change to resolved.
func (e *Engine) resolve() {
	if nil != e.unresolved && len(e.unresolved) > 0 {
		leftover := make([]*hook, 0)
		for _, v := range e.unresolved {
			// make sure the status is unresolved
			if !v.Resolved {
				// find extension point this extension anchors to
				achrs := e.anchors[v.Anchor]
				if nil != achrs && len(achrs) > 0 {
					for _, achr := range achrs {
						if v.Anchor == achr.Id {
							achr.Hooks = append(achr.Hooks, v)
							v.Resolved = true
							e.hooks[v.Id] = v
						} else {
							// not found, append to leftover
							leftover = append(leftover, v)
						}
					}
				}
			}
		}

		// set the leftover unresolved
		e.unresolved = leftover
	}
}

// RegisterHostExtensionPoint
//
// This method allows a host/client application that is using the Plugin Engine to register extension points. This is
// useful if the host/client app has some specific things it wants to allow anchor points for plugins to attach to.
// Ideally a host/client app may ship/install/start with plugins already, but this gives the ability for the host/client
// to have native code functions tied to extension points that are then filled by plugin extensions.
func (e *Engine) RegisterHostExtensionPoint(id, name, version, description string) {
	ep := &anchor{
		Anchor: Anchor{
			Id:          id,
			Description: description,
			Name:        name,
		},
	}

	achrs := e.anchors[id]
	if nil == achrs {
		achrs = make([]*anchor, 0)
	}

	achrs = append(achrs, ep)
	// reassign because exps may be a new larger ref.. has to be reassigned
	e.anchors[id] = achrs
	e.resolve()
}

func (e *Engine) GetPlugins() map[string]map[string]*plugin {
	return e.plugins
}

func (e *Engine) CallHookFunc(hookId string, data []byte) ([]byte, error) {
	callable := callableHooks[hookId]

	if nil != callable {
		extension := e.hooks[hookId]
		if nil == callable.Plugin {
			fmt.Println("Instantiating plugin: ", hookId)
			if err := e.instantiate(callable); err != nil {
				fmt.Println("Problem instantiating callable plugin: ", extension.Func)
			}
		}

		_, d, err := callable.Plugin.Call(extension.Func, data)
		if nil != err {
			return nil, err
		}

		return d, nil
	}

	return nil, nil
}

func NewPluginEngine(hostFuncs []extism.HostFunction, pluginOutputPath string) (*Engine, error) {
	return NewPluginEngineWithLogging(hostFuncs, extism.LogLevelDebug, pluginOutputPath)
}

// NewPluginEngine
//
// This function will create a new plugin engine instance. Passed in are host functions per the Extism (WASI)
// Host Function spec. This allows consumers of this engine to provide its own host functions that plugins will be
// able to utilize along with the plugin engine host functions.
func NewPluginEngineWithLogging(hostFuncs []extism.HostFunction, logLevel extism.LogLevel, pluginOutputPath string) (*Engine, error) {
	plugins := make(map[string]map[string]*plugin)
	unresolved := make([]*hook, 0)
	anchors := make(map[string][]*anchor)
	hooks := make(map[string]*hook)

	// verify that the pluginPath exists and/or if not created.. is created
	err := os.MkdirAll(pluginOutputPath, 0660)
	if err != nil {
		return nil, errors.New("a problem trying to create the plugin output path (" + pluginOutputPath + ") : " + err.Error())
	}

	// instantiate as we need this in the host functions
	engine := &Engine{
		context:    context.Background(),
		logLevel:   logLevel,
		plugins:    plugins,
		unresolved: unresolved,
		hooks:      hooks,
		anchors:    anchors,
		pluginPath: pluginOutputPath,
	}

	hfs := append(hostFuncs, engine.GetHostFuncs()...)
	engine.hostFuncs = hfs

	return engine, nil
}
