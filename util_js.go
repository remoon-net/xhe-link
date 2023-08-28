package main

import (
	"encoding/json"
	"syscall/js"
)

func getConfig[T any](v js.Value) (c T, err error) {
	s := js.Global().Get("JSON").Call("stringify", v).String()
	err = json.Unmarshal([]byte(s), &c)
	return
}
