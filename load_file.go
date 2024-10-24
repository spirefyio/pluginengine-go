package pluginengine

import (
	pdk "github.com/extism/go-pdk"
)

//go:wasmimport extism:host/pluginengine LoadFile
func loadFile(path uint64) uint64

// LoadFile
//
// This function can be called by plugins to call one (or more) extensions matching the extension point id and version.
// If the slice of extensionsIds is nil or empty, the first extension found is called if the extension point match is
// found.
//
// This is a wrapper function which uses the imported callExtensionPointExtensions function implemented in the
// pluginengine plugin. This wrapper makes it easier for Go plugin developers to avoid the WASM memory management
func LoadFile(path string) ([]byte, error) {
	pdk.Log(pdk.LogDebug, "LoadFile")

	// allocate the memory for the string
	dta1 := pdk.AllocateString(path)
	// get the offset
	off1 := dta1.Offset()

	// call the imported host function with the off1 and get its response offset
	resp := loadFile(off1)

	// returned from the call to the imported readFile, so lets grab its memory that stores the file data
	mem1 := pdk.FindMemory(resp)

	// get the actual []bytes
	filedata := mem1.ReadBytes()

	// all done return it
	// TODO: Do we need to free it here.. above with a defer() or let the calling plugin free it somehow?
	return filedata, nil
}
