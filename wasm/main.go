package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"syscall/js"

	gabs "github.com/Jeffail/gabs/v2"
	"gopkg.in/yaml.v3"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

func updateCheckboxes(this js.Value, args []js.Value) interface{} {
	document := js.Global().Get("document")

	targetLinux := document.Call("getElementById", "targetSystem.linux").Get("checked").Bool()
	targetWindows := document.Call("getElementById", "targetSystem.windows").Get("checked").Bool()
	logsEnabled := document.Call("getElementById", "datadog.logs.enabled").Get("checked").Bool()
	apmEnabled := document.Call("getElementById", "datadog.apm.enabled").Get("checked").Bool()
	processAgentEnabled := document.Call("getElementById", "datadog.processAgent.enabled").Get("checked").Bool()
	networkMonitoringEnabled := document.Call("getElementById", "datadog.networkMonitoring.enabled").Get("checked").Bool()
	complianceEnabled := document.Call("getElementById", "datadog.securityAgent.compliance.enabled").Get("checked").Bool()
	runtimeEnabled := document.Call("getElementById", "datadog.securityAgent.runtime.enabled").Get("checked").Bool()

	valuesStr := document.Call("getElementById", "values.yaml").Get("value").String()

	values := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(valuesStr), &values)
	if err != nil {
		fmt.Printf("Failed to unmarshal: %s\n", err)
		return nil
	}

	c := gabs.Wrap(values)
	if targetLinux {
		_, _ = c.Set("linux", "targetSystem")
	} else if targetWindows {
		_, _ = c.Set("windows", "targetSystem")
	}
	_, _ = c.Set(logsEnabled, "datadog", "logs", "enabled")
	_, _ = c.Set(apmEnabled, "datadog", "apm", "enabled")
	_, _ = c.Set(processAgentEnabled, "datadog", "processAgent", "enabled")
	_, _ = c.Set(networkMonitoringEnabled, "datadog", "networkMonitoring", "enabled")
	_, _ = c.Set(complianceEnabled, "datadog", "securityAgent", "compliance", "enabled")
	_, _ = c.Set(runtimeEnabled, "datadog", "securityAgent", "runtime", "enabled")
	values = c.Data().(map[string]interface{})

	if valuesBytes, err := yaml.Marshal(&values); err == nil {
		document.Call("getElementById", "values.yaml").Set("value", string(valuesBytes))
	}

	render()

	return nil
}

func updateValuesYaml(this js.Value, args []js.Value) interface{} {
	document := js.Global().Get("document")
	valuesStr := document.Call("getElementById", "values.yaml").Get("value").String()

	values := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(valuesStr), &values); err != nil {
		document.Call("getElementById", "values-errors").Set("textContent", err.Error())
	} else {
		document.Call("getElementById", "values-errors").Set("textContent", "")

		c := gabs.Wrap(values)
		if value, ok := c.Search("targetSystem").Data().(string); ok {
			switch value {
			case "linux":
				document.Call("getElementById", "targetSystem.linux").Set("checked", true)
			case "windows":
				document.Call("getElementById", "targetSystem.windows").Set("checked", true)
			}
		}
		if value, ok := c.Search("datadog", "logs", "enabled").Data().(bool); ok {
			document.Call("getElementById", "datadog.logs.enabled").Set("checked", value)
		}
		if value, ok := c.Search("datadog", "apm", "enabled").Data().(bool); ok {
			document.Call("getElementById", "datadog.apm.enabled").Set("checked", value)
		}
		if value, ok := c.Search("datadog", "processAgent", "enabled").Data().(bool); ok {
			document.Call("getElementById", "datadog.processAgent.enabled").Set("checked", value)
		}
		if value, ok := c.Search("datadog", "networkMonitoring", "enabled").Data().(bool); ok {
			document.Call("getElementById", "datadog.networkMonitoring.enabled").Set("checked", value)
		}
		if value, ok := c.Search("datadog", "securityAgent", "compliance", "enabled").Data().(bool); ok {
			document.Call("getElementById", "datadog.securityAgent.compliance.enabled").Set("checked", value)
		}
		if value, ok := c.Search("datadog", "securityAgent", "runtime", "enabled").Data().(bool); ok {
			document.Call("getElementById", "datadog.securityAgent.runtime.enabled").Set("checked", value)
		}

		render()
	}

	return nil
}

func render() {
	document := js.Global().Get("document")
	valuesStr := document.Call("getElementById", "values.yaml").Get("value").String()

	chartTar, err := base64.StdEncoding.DecodeString(
		js.Global().Get("chart").String(),
	)
	if err != nil {
		fmt.Printf("Failed to base64 decode the chart tarball: %s\n", err)
		document.Call("getElementById", "rendered_chart").Set("value", err.Error())
		return
	}

	chart, err := loader.LoadArchive(bytes.NewReader(chartTar))
	if err != nil {
		fmt.Printf("Failed to load archive: %s\n", err)
		document.Call("getElementById", "rendered_chart").Set("value", err.Error())
		return
	}

	values, err := chartutil.ReadValues([]byte(valuesStr))
	if err != nil {
		fmt.Printf("Failed to read values: %s\n", err)
		document.Call("getElementById", "rendered_chart").Set("value", err.Error())
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
		document.Call("getElementById", "rendered_chart").Set("value", err.Error())
		return
	}

	rendered, err := engine.Render(chart, values)
	if err != nil {
		fmt.Printf("Failed to render chart: %s\n", err)
		document.Call("getElementById", "rendered_chart").Set("value", err.Error())
		return
	}

	yamlSeparator := "\n---\n"
	notesSeparator := "\n------------------------------------------------\n"
	var manifests, notes string
	for k, v := range rendered {
		if len(v) <= 1 {
			// Ignore empty files
			continue
		}
		if strings.HasSuffix(k, "NOTES.txt") {
			notes += notesSeparator + v
			continue
		}
		manifests += yamlSeparator + v
	}

	manifests = strings.TrimPrefix(manifests, yamlSeparator)
	notes = strings.TrimPrefix(notes, notesSeparator)
	document.Call("getElementById", "rendered_chart").Set("textContent", manifests)
	document.Call("getElementById", "rendered_notes").Set("textContent", notes)

	url := js.Global().Get("URL")
	blob := js.Global().Get("Blob")

	prevObjURL := document.Call("getElementById", "download").Get("href")
	obj := blob.New([]interface{}{manifests}, map[string]interface{}{"type": "text/vnd.yaml"})
	newObjURL := url.Call("createObjectURL", obj)
	document.Call("getElementById", "download").Set("href", newObjURL)
	url.Call("revokeObjectURL", prevObjURL)
}

func registerCallbacks() {
	document := js.Global().Get("document")
	document.Call("getElementById", "targetSystem.linux").Call("addEventListener", "change", js.FuncOf(updateCheckboxes))
	document.Call("getElementById", "targetSystem.windows").Call("addEventListener", "change", js.FuncOf(updateCheckboxes))
	document.Call("getElementById", "datadog.logs.enabled").Call("addEventListener", "change", js.FuncOf(updateCheckboxes))
	document.Call("getElementById", "datadog.apm.enabled").Call("addEventListener", "change", js.FuncOf(updateCheckboxes))
	document.Call("getElementById", "datadog.processAgent.enabled").Call("addEventListener", "change", js.FuncOf(updateCheckboxes))
	document.Call("getElementById", "datadog.networkMonitoring.enabled").Call("addEventListener", "change", js.FuncOf(updateCheckboxes))
	document.Call("getElementById", "datadog.securityAgent.compliance.enabled").Call("addEventListener", "change", js.FuncOf(updateCheckboxes))
	document.Call("getElementById", "datadog.securityAgent.runtime.enabled").Call("addEventListener", "change", js.FuncOf(updateCheckboxes))
	document.Call("getElementById", "values.yaml").Call("addEventListener", "change", js.FuncOf(updateValuesYaml))
}

func main() {
	c := make(chan struct{})

	fmt.Println("WASM Go Initialized")
	registerCallbacks()
	render()
	<-c
}
