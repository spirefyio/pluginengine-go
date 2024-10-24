package pluginengine

import (
	"encoding/json"
	pdk "github.com/extism/go-pdk"
)

//go:wasmimport extism:host/pluginengine GetHooksForAnchor
func getHooks(path uint64) uint64

// LoadFile
//
// This function can be called by plugins to call one (or more) extensions matching the extension point id and version.
// If the slice of extensionsIds is nil or empty, the first extension found is called if the extension point match is
// found.
//
// This is a wrapper function which uses the imported callExtensionPointExtensions function implemented in the
// pluginengine plugin. This wrapper makes it easier for Go plugin developers to avoid the WASM memory management
func GetHooksForAnchor(anchorId string) ([]Hook, error) {
	pdk.Log(pdk.LogDebug, "GetHooksForAnchors")

	// allocate the memory for the string
	dta1 := pdk.AllocateString(anchorId)

	// get the offset
	off1 := dta1.Offset()

	// call the imported host function with the off1 and get its response offset
	resp := getHooks(off1)

	// returned from the call to the imported readFile, so lets grab its memory that stores the file data
	mem1 := pdk.FindMemory(resp)

	// get the actual []bytes
	filedata := mem1.ReadBytes()
	hooks := make([]Hook, 0)

	if nil != filedata && len(filedata) > 0 {
		// marshal back in to the hooks object
		err := json.Unmarshal(filedata, &hooks)
		if nil != err {
			pdk.Log(pdk.LogDebug, "Error converting extension data back to extension object "+err.Error())
			return nil, err
		}
	}

	return hooks, nil
}
