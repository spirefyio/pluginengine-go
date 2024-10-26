package pluginengine

import (
	"context"
	"encoding/json"
	"fmt"
	extism "github.com/extism/go-sdk"
	"io/fs"
	"os"
	"path/filepath"
)

// This Host function allows a plugin to load a local file via the engine. It will load the local file
// as a []byte and pass that directly to the calling plugin as part of the response. It is up to the
// calling plugin to then handle the file contents as needed.
func load(e Engine) extism.HostFunction {
	ret := extism.NewHostFunctionWithStack(
		"LoadFile",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			filePath, err2 := p.ReadString(stack[0])

			if nil != err2 {
				// TODO: Figure out how to handle this correctly
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST getExtensionsForExtensionPoint FUNCTION: ", err2)
			}

			fmt.Println("Plugin is calling LoadFile with path: ", filePath)
			dir := filepath.Dir(filePath)
			filename := filepath.Base(filePath)
			fsys := os.DirFS(dir)

			// Read file contents using fs.ReadFile
			fileData, err := fs.ReadFile(fsys, filename)
			if err != nil {
				// TODO: LOG THIS ERROR SOMEHOW
				fmt.Println("Problem reading file: ", err)
			}

			// write it back out to the calling plugin, so it can get it as a response to the host func call
			ff, err := p.WriteBytes(fileData)
			stack[0] = ff

			if err != nil {
				fmt.Println("Error writing bytes: ", err)
			}
		},
		[]extism.ValueType{extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
	)
	ret.SetNamespace("extism:host/pluginengine")

	return ret
}

// This function allows plugin anchor code or other hook code to call a hook function. The hook
// function can reside in any loaded resolved plugin. It utilizes the Extism/WASM memory stack
// to pass in parameters expected by the anchor the hook is tied in to.
func hookCall(e Engine) extism.HostFunction {
	ret := extism.NewHostFunctionWithStack(
		"CallHook",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			hkId, err := p.ReadString(stack[0])

			if nil != err {
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST getExtensionsForExtensionPoint FUNCTION: ", err)
			}

			fmt.Println("Calling CallHook from plugin for hook id: ", hkId)

			data, err := p.ReadBytes(stack[1])

			if nil != err {
				fmt.Println("ERROR READING BYTES OF INPUT DATA")
			}

			if nil != data && len(data) > 0 {
				fmt.Println("WE GOT DATA.. it should be passed on to the extension to be called")
			}

			extResp, err := e.CallHookFunc(hkId, data)
			if nil != err {
				fmt.Println("ERROR IN HOST FUNC: ", err)
			}

			if nil != extResp {
				ff, err := p.WriteBytes(extResp)

				if err != nil {
					fmt.Println("Error writing bytes: ", err)
					return
				} else {
					stack[0] = ff
				}
			}

		},
		[]extism.ValueType{extism.ValueTypeI64, extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
	)
	ret.SetNamespace("extism:host/pluginengine")

	return ret
}

func hooksForAnchor(e Engine) extism.HostFunction {
	ret := extism.NewHostFunctionWithStack(
		"GetHooks",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			// Grab the extension point from memory/stack
			extPtId, err := p.ReadString(stack[0])

			if nil != err {
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST getExtensions FUNCTION: ", err)
			}

			fmt.Println("Calling GetExtensions from plugin for extensionPoint id: ", extPtId)

			hooks, err := e.GetHooksForAnchor(extPtId)

			if nil != err {
				fmt.Println("ERROR IN HOST FUNC: ", err)
			}

			if nil != hooks && len(hooks) > 0 {
				// marshal the objects into jsonBytes
				jsonBytes, err := json.Marshal(hooks)

				ff, err := p.WriteBytes(jsonBytes)

				if err != nil {
					fmt.Println("Error writing bytes: ", err)
					return
				} else {
					stack[0] = ff
				}
			} else {
				// no bytes.. set stack to 0
				stack[0] = 0
			}
		},
		[]extism.ValueType{extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
	)
	ret.SetNamespace("extism:host/pluginengine")

	return ret
}

func (e Engine) GetHostFuncs() []extism.HostFunction {
	return []extism.HostFunction{hookCall(e), load(e), hooksForAnchor(e)}
}

