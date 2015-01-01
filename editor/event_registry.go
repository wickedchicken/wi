// generated by go run ../tools/wi-event-generator/main.go -output event_registry.go -impl; DO NOT EDIT

package editor

import (
	"errors"
	"sync"

	"github.com/maruel/wi/pkg/key"
	"github.com/maruel/wi/wicore"
)

type eventCommands struct {
	id       wicore.EventID
	callback func(a wicore.EnqueuedCommands) bool
}

type eventDocumentCreated struct {
	id       wicore.EventID
	callback func(a wicore.Document) bool
}

type eventDocumentCursorMoved struct {
	id       wicore.EventID
	callback func(a wicore.Document, b int, c int) bool
}

type eventTerminalKeyPressed struct {
	id       wicore.EventID
	callback func(a key.Press) bool
}

type eventTerminalResized struct {
	id       wicore.EventID
	callback func() bool
}

type eventViewCreated struct {
	id       wicore.EventID
	callback func(a wicore.View) bool
}

type eventWindowCreated struct {
	id       wicore.EventID
	callback func(a wicore.Window) bool
}

type eventWindowResized struct {
	id       wicore.EventID
	callback func(a wicore.Window) bool
}

// eventRegistry is automatically generated via wi-event-generator from the
// interface wicore.EventRegistry.
type eventRegistry struct {
	lock     sync.Mutex
	nextID   wicore.EventID
	deferred chan func()

	commands            []eventCommands
	documentCreated     []eventDocumentCreated
	documentCursorMoved []eventDocumentCursorMoved
	terminalKeyPressed  []eventTerminalKeyPressed
	terminalResized     []eventTerminalResized
	viewCreated         []eventViewCreated
	windowCreated       []eventWindowCreated
	windowResized       []eventWindowResized
}

func makeEventRegistry() eventRegistry {
	// Reduce the odds of allocation within RegistryXXX() by using relatively
	// large buffers.
	return eventRegistry{
		deferred:            make(chan func(), 2048),
		commands:            make([]eventCommands, 0, 64),
		documentCreated:     make([]eventDocumentCreated, 0, 64),
		documentCursorMoved: make([]eventDocumentCursorMoved, 0, 64),
		terminalKeyPressed:  make([]eventTerminalKeyPressed, 0, 64),
		terminalResized:     make([]eventTerminalResized, 0, 64),
		viewCreated:         make([]eventViewCreated, 0, 64),
		windowCreated:       make([]eventWindowCreated, 0, 64),
		windowResized:       make([]eventWindowResized, 0, 64),
	}
}

func (er *eventRegistry) Unregister(eventID wicore.EventID) error {
	er.lock.Lock()
	defer er.lock.Unlock()
	// TODO(maruel): The buffers are never reallocated, so it's effectively a
	// memory leak.
	switch eventID & wicore.EventID(0xff000000) {
	case wicore.EventID(0x1000000):
		for index, value := range er.commands {
			if value.id == eventID {
				copy(er.commands[index:], er.commands[index+1:])
				er.commands = er.commands[0 : len(er.commands)-1]
				return nil
			}
		}
	case wicore.EventID(0x2000000):
		for index, value := range er.documentCreated {
			if value.id == eventID {
				copy(er.documentCreated[index:], er.documentCreated[index+1:])
				er.documentCreated = er.documentCreated[0 : len(er.documentCreated)-1]
				return nil
			}
		}
	case wicore.EventID(0x3000000):
		for index, value := range er.documentCursorMoved {
			if value.id == eventID {
				copy(er.documentCursorMoved[index:], er.documentCursorMoved[index+1:])
				er.documentCursorMoved = er.documentCursorMoved[0 : len(er.documentCursorMoved)-1]
				return nil
			}
		}
	case wicore.EventID(0x4000000):
		for index, value := range er.terminalKeyPressed {
			if value.id == eventID {
				copy(er.terminalKeyPressed[index:], er.terminalKeyPressed[index+1:])
				er.terminalKeyPressed = er.terminalKeyPressed[0 : len(er.terminalKeyPressed)-1]
				return nil
			}
		}
	case wicore.EventID(0x5000000):
		for index, value := range er.terminalResized {
			if value.id == eventID {
				copy(er.terminalResized[index:], er.terminalResized[index+1:])
				er.terminalResized = er.terminalResized[0 : len(er.terminalResized)-1]
				return nil
			}
		}
	case wicore.EventID(0x6000000):
		for index, value := range er.viewCreated {
			if value.id == eventID {
				copy(er.viewCreated[index:], er.viewCreated[index+1:])
				er.viewCreated = er.viewCreated[0 : len(er.viewCreated)-1]
				return nil
			}
		}
	case wicore.EventID(0x7000000):
		for index, value := range er.windowCreated {
			if value.id == eventID {
				copy(er.windowCreated[index:], er.windowCreated[index+1:])
				er.windowCreated = er.windowCreated[0 : len(er.windowCreated)-1]
				return nil
			}
		}
	case wicore.EventID(0x8000000):
		for index, value := range er.windowResized {
			if value.id == eventID {
				copy(er.windowResized[index:], er.windowResized[index+1:])
				er.windowResized = er.windowResized[0 : len(er.windowResized)-1]
				return nil
			}
		}
	}
	return errors.New("trying to unregister an non existing event listener")
}

func (er *eventRegistry) RegisterCommands(callback func(a wicore.EnqueuedCommands) bool) wicore.EventID {
	er.lock.Lock()
	defer er.lock.Unlock()
	i := er.nextID
	er.nextID++
	er.commands = append(er.commands, eventCommands{i, callback})
	return i | wicore.EventID(0x1000000)
}

func (er *eventRegistry) onCommands(a wicore.EnqueuedCommands) {
	er.deferred <- func() {
		items := func() []func(a wicore.EnqueuedCommands) bool {
			er.lock.Lock()
			defer er.lock.Unlock()
			items := make([]func(a wicore.EnqueuedCommands) bool, 0, len(er.commands))
			for _, item := range er.commands {
				items = append(items, item.callback)
			}
			return items
		}()
		for _, item := range items {
			if !item(a) {
				break
			}
		}
	}
}

func (er *eventRegistry) RegisterDocumentCreated(callback func(a wicore.Document) bool) wicore.EventID {
	er.lock.Lock()
	defer er.lock.Unlock()
	i := er.nextID
	er.nextID++
	er.documentCreated = append(er.documentCreated, eventDocumentCreated{i, callback})
	return i | wicore.EventID(0x2000000)
}

func (er *eventRegistry) onDocumentCreated(a wicore.Document) {
	er.deferred <- func() {
		items := func() []func(a wicore.Document) bool {
			er.lock.Lock()
			defer er.lock.Unlock()
			items := make([]func(a wicore.Document) bool, 0, len(er.documentCreated))
			for _, item := range er.documentCreated {
				items = append(items, item.callback)
			}
			return items
		}()
		for _, item := range items {
			if !item(a) {
				break
			}
		}
	}
}

func (er *eventRegistry) RegisterDocumentCursorMoved(callback func(a wicore.Document, b int, c int) bool) wicore.EventID {
	er.lock.Lock()
	defer er.lock.Unlock()
	i := er.nextID
	er.nextID++
	er.documentCursorMoved = append(er.documentCursorMoved, eventDocumentCursorMoved{i, callback})
	return i | wicore.EventID(0x3000000)
}

func (er *eventRegistry) onDocumentCursorMoved(a wicore.Document, b int, c int) {
	er.deferred <- func() {
		items := func() []func(a wicore.Document, b int, c int) bool {
			er.lock.Lock()
			defer er.lock.Unlock()
			items := make([]func(a wicore.Document, b int, c int) bool, 0, len(er.documentCursorMoved))
			for _, item := range er.documentCursorMoved {
				items = append(items, item.callback)
			}
			return items
		}()
		for _, item := range items {
			if !item(a, b, c) {
				break
			}
		}
	}
}

func (er *eventRegistry) RegisterTerminalKeyPressed(callback func(a key.Press) bool) wicore.EventID {
	er.lock.Lock()
	defer er.lock.Unlock()
	i := er.nextID
	er.nextID++
	er.terminalKeyPressed = append(er.terminalKeyPressed, eventTerminalKeyPressed{i, callback})
	return i | wicore.EventID(0x4000000)
}

func (er *eventRegistry) onTerminalKeyPressed(a key.Press) {
	er.deferred <- func() {
		items := func() []func(a key.Press) bool {
			er.lock.Lock()
			defer er.lock.Unlock()
			items := make([]func(a key.Press) bool, 0, len(er.terminalKeyPressed))
			for _, item := range er.terminalKeyPressed {
				items = append(items, item.callback)
			}
			return items
		}()
		for _, item := range items {
			if !item(a) {
				break
			}
		}
	}
}

func (er *eventRegistry) RegisterTerminalResized(callback func() bool) wicore.EventID {
	er.lock.Lock()
	defer er.lock.Unlock()
	i := er.nextID
	er.nextID++
	er.terminalResized = append(er.terminalResized, eventTerminalResized{i, callback})
	return i | wicore.EventID(0x5000000)
}

func (er *eventRegistry) onTerminalResized() {
	er.deferred <- func() {
		items := func() []func() bool {
			er.lock.Lock()
			defer er.lock.Unlock()
			items := make([]func() bool, 0, len(er.terminalResized))
			for _, item := range er.terminalResized {
				items = append(items, item.callback)
			}
			return items
		}()
		for _, item := range items {
			if !item() {
				break
			}
		}
	}
}

func (er *eventRegistry) RegisterViewCreated(callback func(a wicore.View) bool) wicore.EventID {
	er.lock.Lock()
	defer er.lock.Unlock()
	i := er.nextID
	er.nextID++
	er.viewCreated = append(er.viewCreated, eventViewCreated{i, callback})
	return i | wicore.EventID(0x6000000)
}

func (er *eventRegistry) onViewCreated(a wicore.View) {
	er.deferred <- func() {
		items := func() []func(a wicore.View) bool {
			er.lock.Lock()
			defer er.lock.Unlock()
			items := make([]func(a wicore.View) bool, 0, len(er.viewCreated))
			for _, item := range er.viewCreated {
				items = append(items, item.callback)
			}
			return items
		}()
		for _, item := range items {
			if !item(a) {
				break
			}
		}
	}
}

func (er *eventRegistry) RegisterWindowCreated(callback func(a wicore.Window) bool) wicore.EventID {
	er.lock.Lock()
	defer er.lock.Unlock()
	i := er.nextID
	er.nextID++
	er.windowCreated = append(er.windowCreated, eventWindowCreated{i, callback})
	return i | wicore.EventID(0x7000000)
}

func (er *eventRegistry) onWindowCreated(a wicore.Window) {
	er.deferred <- func() {
		items := func() []func(a wicore.Window) bool {
			er.lock.Lock()
			defer er.lock.Unlock()
			items := make([]func(a wicore.Window) bool, 0, len(er.windowCreated))
			for _, item := range er.windowCreated {
				items = append(items, item.callback)
			}
			return items
		}()
		for _, item := range items {
			if !item(a) {
				break
			}
		}
	}
}

func (er *eventRegistry) RegisterWindowResized(callback func(a wicore.Window) bool) wicore.EventID {
	er.lock.Lock()
	defer er.lock.Unlock()
	i := er.nextID
	er.nextID++
	er.windowResized = append(er.windowResized, eventWindowResized{i, callback})
	return i | wicore.EventID(0x8000000)
}

func (er *eventRegistry) onWindowResized(a wicore.Window) {
	er.deferred <- func() {
		items := func() []func(a wicore.Window) bool {
			er.lock.Lock()
			defer er.lock.Unlock()
			items := make([]func(a wicore.Window) bool, 0, len(er.windowResized))
			for _, item := range er.windowResized {
				items = append(items, item.callback)
			}
			return items
		}()
		for _, item := range items {
			if !item(a) {
				break
			}
		}
	}
}
