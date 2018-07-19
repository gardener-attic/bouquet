/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package engine

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

// Engine is an implementation of 'cmd/tiller/environment'.Engine that uses Go templates.
type Engine struct {
	// FuncMap contains the template functions that will be passed to each
	// render call. This may only be modified before the first call to Render.
	FuncMap template.FuncMap
}

// New creates a new Go template Engine instance.
//
// The FuncMap is initialized here. You may modify the FuncMap _prior to_ the
// first invocation of Render.
//
// The FuncMap sets all of the Sprig functions except for those that provide
// access to the underlying OS (env, expandenv).
func New() *Engine {
	f := sprig.TxtFuncMap()
	delete(f, "env")
	delete(f, "expandenv")
	return &Engine{
		FuncMap: f,
	}
}

// Render takes a chart, optional values, and value overrids, and attempts to render the Go templates.
//
// Render can be called repeatedly on the same engine.
//
// This will look in the chart's 'templates' data (e.g. the 'templates/' directory)
// and attempt to render the templates there using the values passed in.
//
// Values are scoped to their templates. A dependency template will not have
// access to the values set for its parent. If chart "foo" includes chart "bar",
// "bar" will not have access to the values for "foo".
//
// Values should be prepared with something like `chartutils.ReadValues`.
//
// Values are passed through the templates according to scope. If the top layer
// chart includes the chart foo, which includes the chart bar, the values map
// will be examined for a table called "foo". If "foo" is found in vals,
// that section of the values will be passed into the "foo" chart. And if that
// section contains a value named "bar", that value will be passed on to the
// bar chart during render time.
func (e *Engine) Render(chrt *chart.Chart, values chartutil.Values) (map[string]string, error) {
	// Render the charts
	tmap := allTemplates(chrt, values)
	return e.render(tmap)
}

// renderable is an object that can be rendered.
type renderable struct {
	// tpl is the current template.
	tpl string
	// vals are the values to be supplied to the template.
	vals chartutil.Values
}

// render takes a map of templates/values and renders them.
func (e *Engine) render(tpls map[string]renderable) (map[string]string, error) {
	// Basically, what we do here is start with an empty parent template and then
	// build up a list of templates -- one for each file. Once all of the templates
	// have been parsed, we loop through again and execute every template.
	//
	// The idea with this process is to make it possible for more complex templates
	// to share common blocks, but to make the entire thing feel like a file-based
	// template engine.
	t := template.New("gotpl")
	files := []string{}
	for fname, r := range tpls {
		t = t.New(fname).Funcs(e.FuncMap)
		if _, err := t.Parse(r.tpl); err != nil {
			return map[string]string{}, fmt.Errorf("parse error in %q: %s", fname, err)
		}
		files = append(files, fname)
	}

	rendered := make(map[string]string, len(files))
	var buf bytes.Buffer
	for _, file := range files {
		//	log.Printf("Exec %s with %v (%s)", file, tpls[file].vals, tpls[file].tpl)
		if err := t.ExecuteTemplate(&buf, file, tpls[file].vals); err != nil {
			return map[string]string{}, fmt.Errorf("render error in %q: %s", file, err)
		}
		rendered[file] = buf.String()
		buf.Reset()
	}

	return rendered, nil
}

// allTemplates returns all templates for a chart and its dependencies.
//
// As it goes, it also prepares the values in a scope-sensitive manner.
func allTemplates(c *chart.Chart, vals chartutil.Values) map[string]renderable {
	templates := map[string]renderable{}
	recAllTpls(c, templates, vals, true)
	return templates
}

// recAllTpls recurses through the templates in a chart.
//
// As it recurses, it also sets the values to be appropriate for the template
// scope.
func recAllTpls(c *chart.Chart, templates map[string]renderable, parentVals chartutil.Values, top bool) {
	var cvals chartutil.Values
	if top {
		// If this is the top of the rendering tree, assume that parentVals
		// is already resolved to the authoritative values.
		cvals = parentVals
	} else if c.Metadata != nil && c.Metadata.Name != "" {
		// An error indicates that the table doesn't exist. So we leave it as
		// an empty map.
		tmp, err := parentVals.Table(c.Metadata.Name)
		if err == nil {
			cvals = tmp
		}
	}

	//log.Printf("racAllTpls values: %v", cvals)
	for _, child := range c.Dependencies {
		recAllTpls(child, templates, cvals, false)
	}
	for _, t := range c.Templates {
		templates[t.Name] = renderable{
			tpl:  string(t.Data),
			vals: cvals,
		}
	}
}
