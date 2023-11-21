package main

import (
	"strings"
	"syscall/js"

	"github.com/buildkite/go-pipeline"
)

func main() {
	c := make(chan struct{})

	js.Global().Set("parseYAML", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		input := args[0].String()

		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]

			go func() {
				p, err := pipeline.Parse(strings.NewReader(input))
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
				}

				json, err := p.MarshalJSON()
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
				}

				output := js.ValueOf(js.Global().Get("JSON").Call("parse", string(json)))
				resolve.Invoke(js.ValueOf(output))
			}()

			return nil
		})

		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	}))

	<-c
}
