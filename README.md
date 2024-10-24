# SpirefyIO PluginEngine Go Implementation

This is a plugin engine/that uses the go Extism project SDK and allows a Go application to load this library to enable plugin support via Extism dependent WASM plugin modules written in many languages. A separate library/repo that compliments this project for Go plugin development is the plugin-go-pdk (github.com/spirefyio/plugin-go-pdk) which allows plugin developers to make use of plugin engine exported host functions.

The engine itself manages the loading of plugins, processing their plugin configuration (plugin.yaml) files and building an in memory structure of plugin anchors, hooks, events, listeners, and more. 

Anchor:
  This is defined in the plugin.yaml config file that is bundled with a plugin .wasm module in an archive file structure (usually .zip). An Anchor is a place holder where plugins can contribute code to extend the functionality at that anchor point. Plugins can define as many anchors as they desire. Code within the plugin will use the anchor points to find any resolved hooks (see below) and do something with those hooks.. call them usually within some context. Each anchor should be well defined, including any payload the anchor sends to the hook and any response expected in return from the hook implementation. These should typically be provided as something like a JSON Schema so that plugin hook authors can generate a proper structure of code in whatever languge they are writing the hook plugin in. Or a description of the expected structure.

Hook:
  A hook attaches (resolves to) an anchor and is configured in the plugin.yaml configuration file. A plugin can contribute hooks to multiple anchors or even multiple hooks to one anchor if there is a reason to do so. For example a single plugin may provide multipel file extension hooks to a file dialog anchor. Hooks provide an exported WASM function and typically accept a payload defined by the anchor if provided, and response with a payload if defined. 

  It's important to understand that WASM does NOT support references, so any data between an anchor plugin and a hook plugin must serialize from the anchor to the hook, and likewise any response would be serialized by the hook plugin and deserialized by the anchor plugin.

Event:
  Events are part of the more generic event bus the engine provides. This is not a fully featured event bus implementation. It's purely so plugins can fire off events and other plugins can respond to them by adding listeners (essentially exported functions) to the events. It is a little more decoupled than the direction anchor and hook approach, but can also allow for asyncronous operations to occur depending on the implementation within the plugin itself. The engine does NOT spawn threads for each event fired, so it is up to the receiving listener plugins to handle threading if desired.

Listener:
  The other end of the event, similar to a hook this would be an expoerted function but would be called by the engine whenever a plugin makes use of SendEvent exported host function.


Plugin versioning is simple. It uses a SemVer x.y.z version value. All anchors and hooks a plugin defines are matched to the version specified. Plugin resolution occurs based on versions. A plugin hook or listener will resolve to a matched anchor or event based on the .z component of the version being any value. If the .y portion is different, this denotes a patch and could be a breaking change. 
(MORE TO COME ON VERSIONING)