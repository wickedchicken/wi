// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Generates editor/event_registry.go from wicore/events.go.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"text/template"

	"github.com/maruel/wi/wicore"
)

func formatSource(buf []byte) ([]byte, error) {
	src, err := format.Source(buf)
	if err != nil {
		b := bytes.Buffer{}
		fmt.Printf("// ERROR: internal error: invalid Go generated: %s\n", err)
		fmt.Printf("// Compile the package to analyze the error.\n\n")
		_, _ = b.Write(buf)
		src = b.Bytes()
	}
	return src, err
}

var tmpl = template.Must(template.New("render").Parse(`// generated by wi-event-generator; DO NOT EDIT

package editor

import (
  "errors"
  "sync"

  "github.com/maruel/wi/pkg/key"
  "github.com/maruel/wi/wicore"
)
{{range .Events}}
type event{{.Name}} struct{
	id wicore.EventID
	callback func({{.Args}})
}
{{end}}
// eventRegistry is automatically generated via wi-event-generator from the
// interface wicore.EventRegistry.
type eventRegistry struct {
  lock   sync.Mutex
  nextID wicore.EventID
	deferred chan func()
{{range .Events}}
	{{.Lower}} []event{{.Name}}{{end}}
}

func makeEventRegistry() eventRegistry {
	// Reduce the odds of allocation within RegistryXXX() by using relatively
	// large buffers.
	return eventRegistry{
		deferred: make(chan func()),{{range .Events}}
		{{.Lower}}: make([]event{{.Name}}, 0, 64),{{end}}
	}
}

func (er *eventRegistry) Unregister(eventID wicore.EventID) error {
  er.lock.Lock()
  defer er.lock.Unlock()
	// TODO(maruel): The buffers are never reallocated, so it's effectively a
	// memory leak.
	switch(eventID & {{.BitMask}}) { {{range .Events}}
	case {{.BitValue}}:
		for index, value := range er.{{.Lower}} {
			if value.id == eventID {
				copy(er.{{.Lower}}[index:], er.{{.Lower}}[index+1:])
				er.{{.Lower}} = er.{{.Lower}}[0 : len(er.{{.Lower}})-1]
				return nil
			}
		}{{end}}
  }
	return errors.New("trying to unregister an non existing event listener")
}{{range .Events}}

func (er *eventRegistry) Register{{.Name}}(callback func({{.Args}})) wicore.EventID {
  er.lock.Lock()
  defer er.lock.Unlock()
  i := er.nextID
  er.nextID++
  er.{{.Lower}} = append(er.{{.Lower}}, event{{.Name}}{i, callback})
  return i | {{.BitValue}}
}

func (er *eventRegistry) on{{.Name}}({{.Args}}) {
  items := func() []func({{.Args}}) {
    er.lock.Lock()
    defer er.lock.Unlock()
    items := make([]func({{.Args}}), 0, len(er.{{.Lower}}))
    for _, item := range er.{{.Lower}} {
      items = append(items, item.callback)
    }
    return items
  }()
  for _, item := range items {
    item({{.ArgsNames}})
  }
}{{end}}
`))

type Event struct {
	Name      string
	Lower     string
	Index     int
	BitValue  string
	Args      string
	ArgsNames string
}

type data struct {
	BitMask string
	Events  []Event
}

func getEvents(bitmask uint) []Event {
	t := reflect.TypeOf((*wicore.EventRegistry)(nil)).Elem()
	events := make([]Event, 0, t.NumMethod())
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		if !strings.HasPrefix(m.Name, "Register") {
			continue
		}
		// That is *very* cheezy. The right way would be to use go/parser to
		// extract the argument names. For now, it's "good enough".
		argsStr := m.Type.String()[10:]
		argsStr = argsStr[:strings.LastIndex(argsStr, ")")-1]
		argsItems := strings.Split(argsStr, ", ")
		args := make([]string, 0, len(argsItems))
		argsNames := make([]string, 0, len(argsItems))
		if len(argsStr) > 0 {
			// Insert names 'a', 'b', ...
			for i, s := range argsItems {
				args = append(args, fmt.Sprintf("%c %s", i+'a', s))
				argsNames = append(argsNames, fmt.Sprintf("%c", i+'a'))
			}
		}
		name := m.Name[8:]
		lower := strings.ToLower(name[0:1]) + name[1:]
		// TODO(maruel): It creates an artificial limit of 2^23 event listener and
		// 2^8 event types on 32 bits systems.
		events = append(events, Event{
			Name:      name,
			Lower:     lower,
			Index:     len(events),
			BitValue:  fmt.Sprintf("wicore.EventID(0x%x)", (len(events)+1)<<bitmask),
			Args:      strings.Join(args, ", "),
			ArgsNames: strings.Join(argsNames, ", ")})
	}
	return events
}

func generate() ([]byte, error) {
	bitmask := uint(24)
	d := data{
		BitMask: fmt.Sprintf("wicore.EventID(0x%x)", ((1<<32)-1)-((1<<bitmask)-1)),
		Events:  getEvents(bitmask),
	}
	out := bytes.Buffer{}
	if err := tmpl.Execute(&out, d); err != nil {
		return nil, fmt.Errorf("failed to generate go code: %s", err)
	}
	return out.Bytes(), nil
}

func mainImpl() int {
	src, err := generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}
	src, err = formatSource(src)
	err2 := ioutil.WriteFile("event_registry.go", src, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}
	if err2 != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err2)
		return 1
	}
	return 0
}

func main() {
	os.Exit(mainImpl())
}
