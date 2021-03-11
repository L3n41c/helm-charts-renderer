package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"syscall/js"

	gabs "github.com/Jeffail/gabs/v2"
	"gopkg.in/yaml.v3"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

func updateCheckboxes(this js.Value, args []js.Value) interface{} {
	document := js.Global().Get("document")

	logsEnabled := document.Call("getElementById", "datadog.logs.enabled").Get("checked").Bool()
	apmEnabled := document.Call("getElementById", "datadog.apm.enabled").Get("checked").Bool()
	processAgentEnabled := document.Call("getElementById", "datadog.processAgent.enabled").Get("checked").Bool()

	valuesStr := document.Call("getElementById", "values.yaml").Get("value").String()

	values := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(valuesStr), &values)
	if err != nil {
		fmt.Printf("Failed to unmarshal: %s\n", err)
		return nil
	}

	c := gabs.Wrap(values)
	c.Set(logsEnabled, "datadog", "logs", "enabled")
	c.Set(apmEnabled, "datadog", "apm", "enabled")
	c.Set(processAgentEnabled, "datadog", "processAgent", "enabled")
	values = c.Data().(map[string]interface{})

	valuesBytes, err := yaml.Marshal(&values)

	document.Call("getElementById", "values.yaml").Set("textContent", string(valuesBytes))

	render()
	return nil
}

func updateValuesYaml(this js.Value, args []js.Value) interface{} {
	render()
	return nil
}

func render() {
	document := js.Global().Get("document")
	valuesStr := document.Call("getElementById", "values.yaml").Get("value").String()

	// resp, err := http.Get("https://github.com/DataDog/helm-charts/releases/download/datadog-2.10.1/datadog-2.10.1.tgz")
	// if err != nil {
	// 	fmt.Printf("Failed to GET the chart archive: %s", err)
	// 	return
	// }
	// defer resp.Body.Close()

	chartTar, err := base64.StdEncoding.DecodeString(
		js.Global().Get("chart").String(),
	)
	if err != nil {
		fmt.Printf("Failed to base64 decode the chart tarball: %s\n", err)
		document.Call("getElementById", "rendered_chart").Set("textContent", err.Error())
		return
	}

	chart, err := loader.LoadArchive(bytes.NewReader(chartTar))
	if err != nil {
		fmt.Printf("Failed to load archive: %s\n", err)
		document.Call("getElementById", "rendered_chart").Set("textContent", err.Error())
		return
	}

	values, err := chartutil.ReadValues([]byte(valuesStr))
	if err != nil {
		fmt.Printf("Failed to read values: %s\n", err)
		document.Call("getElementById", "rendered_chart").Set("textContent", err.Error())
		return
	}

	releaseOptions := chartutil.ReleaseOptions{
		Name:      "datadog",
		Namespace: "default",
		Revision:  1,
		IsUpgrade: false,
		IsInstall: true,
	}
	values, err = chartutil.ToRenderValues(chart, values, releaseOptions, nil)
	if err != nil {
		fmt.Printf("Failed to render values: %s\n", err)
		document.Call("getElementById", "rendered_chart").Set("textContent", err.Error())
		return
	}

	rendered, err := engine.Render(chart, values)
	if err != nil {
		fmt.Printf("Failed to render chart: %s\n", err)
		document.Call("getElementById", "rendered_chart").Set("textContent", err.Error())
		return
	}

	allRendered := ""
	for _, v := range rendered {
		allRendered += v
	}

	document.Call("getElementById", "rendered_chart").Set("textContent", allRendered)
}

func registerCallbacks() {
	js.Global().Set("updateCheckboxes", js.FuncOf(updateCheckboxes))
	js.Global().Set("updateValuesYaml", js.FuncOf(updateValuesYaml))
}

func main() {
	c := make(chan struct{}, 0)

	fmt.Println("WASM Go Initialized")
	registerCallbacks()
	<-c
}
