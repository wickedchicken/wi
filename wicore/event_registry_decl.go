// generated by go run ../tools/wi-event-generator/main.go -output event_registry_decl.go; DO NOT EDIT

package wicore

import (
	"io"

	"github.com/maruel/wi/wicore/key"
	"github.com/maruel/wi/wicore/lang"
)

// EventListener is to be used to cancel an event listener.
type EventListener interface {
	io.Closer
}

// EventRegistry permits to register callbacks that are called on events.
//
// When the callback returns false, the next registered events are not called.
//
// Warning: This interface is automatically generated.
type EventRegistry interface {
	EventsDefinition

	RegisterCommands(callback func(a EnqueuedCommands)) EventListener
	RegisterDocumentCreated(callback func(a Document)) EventListener
	RegisterDocumentCursorMoved(callback func(a Document, b int, c int)) EventListener
	RegisterEditorKeyboardModeChanged(callback func(a KeyboardMode)) EventListener
	RegisterEditorLanguage(callback func(a lang.Language)) EventListener
	RegisterTerminalKeyPressed(callback func(a key.Press)) EventListener
	RegisterTerminalMetaKeyPressed(callback func(a key.Press)) EventListener
	RegisterTerminalResized(callback func()) EventListener
	RegisterViewActivated(callback func(a View)) EventListener
	RegisterViewCreated(callback func(a View)) EventListener
	RegisterWindowCreated(callback func(a Window)) EventListener
	RegisterWindowResized(callback func(a Window)) EventListener
}
