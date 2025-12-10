// Package viatest provides test helpers for Via + gost-dom integration testing.
//
// Via uses always-on SSE connections for live updates, which means standard
// gost-dom ProcessEvents(ctx) will block waiting for SSE to close. This package
// provides helpers that work around this limitation.
//
// Usage:
//
//	func TestMyPage(t *testing.T) {
//		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//		defer cancel()
//
//		v := via.New()
//		// register pages...
//
//		tb := viatest.NewTestBrowser(t, ctx, v)
//		defer tb.Close()
//
//		win := tb.Open("/mypage")
//		tb.Init(win) // Process initial JS/SSE handshake
//
//		// Do assertions...
//
//		tb.WaitFor(win, func() bool {
//			return someCondition()
//		})
//	}
package viatest

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/go-via/via"
	"github.com/gost-dom/browser"
	"github.com/gost-dom/browser/dom"
	"github.com/gost-dom/browser/html"
	"github.com/gost-dom/browser/scripting/v8engine"
	"github.com/gost-dom/browser/testing/gosttest"
)

// TestBrowser wraps a gost-dom browser configured for Via testing.
type TestBrowser struct {
	t       testing.TB
	ctx     context.Context
	browser *browser.Browser
}

// NewTestBrowser creates a browser configured for Via testing.
// It silences Via's logs to prevent test noise.
func NewTestBrowser(t testing.TB, ctx context.Context, v *via.V) *TestBrowser {
	// Silence Via's logs in tests to prevent noise
	SilenceViaLogs()

	b := browser.New(
		browser.WithScriptEngine(v8engine.DefaultEngine()),
		browser.WithContext(ctx),
		browser.WithHandler(v.Handler()),
		browser.WithLogger(gosttest.NewTestingLogger(t)),
	)

	return &TestBrowser{
		t:       t,
		ctx:     ctx,
		browser: b,
	}
}

// Open opens a URL and returns the window.
func (tb *TestBrowser) Open(url string) html.Window {
	win, err := tb.browser.Open(url)
	if err != nil {
		tb.t.Fatalf("failed to open %s: %v", url, err)
	}
	return win
}

// Init processes initial JavaScript and SSE handshake.
// Via renders server-side but needs time for Datastar JS initialization.
// This uses Clock.Advance() which doesn't block on always-on SSE.
func (tb *TestBrowser) Init(win html.Window) {
	_ = win.Clock().Advance(100 * time.Millisecond)
}

// WaitFor waits until the condition function returns false.
// This is the correct way to wait for async actions in Via tests
// because Via's SSE never closes (unlike standard gost-dom ProcessEvents).
//
// The condition function should return true while waiting, false when done.
// Example: wait until mock.actions has entries:
//
//	tb.WaitFor(win, func() bool { return len(mock.actions) == 0 })
func (tb *TestBrowser) WaitFor(win html.Window, stillWaiting func() bool) {
	_ = win.Clock().ProcessEventsWhile(tb.ctx, stillWaiting)
}

// Close closes the browser.
func (tb *TestBrowser) Close() {
	tb.browser.Close()
}

// Browser returns the underlying gost-dom browser for advanced usage.
func (tb *TestBrowser) Browser() *browser.Browser {
	return tb.browser
}

// SilenceViaLogs redirects Via's log output to discard.
// Via logs to the standard log package, so we redirect it.
// Call this before creating Via instances in tests.
func SilenceViaLogs() {
	log.SetOutput(io.Discard)
}

// RestoreLogs restores standard logging (call in defer if needed).
func RestoreLogs(w io.Writer) {
	log.SetOutput(w)
}

// ClickButton finds and clicks a button by text content.
// Returns true if button was found and clicked.
func ClickButton(doc dom.Document, text string) bool {
	buttons := doc.GetElementsByTagName("button")
	for i := 0; i < buttons.Length(); i++ {
		node := buttons.Item(i)
		if btn, ok := node.(html.HTMLElement); ok {
			if btn.TextContent() == text {
				btn.Click()
				return true
			}
		}
	}
	return false
}

// FindElement finds an element by ID and returns it as HTMLElement.
func FindElement(doc dom.Document, id string) html.HTMLElement {
	el := doc.GetElementById(id)
	if el == nil {
		return nil
	}
	if htmlEl, ok := el.(html.HTMLElement); ok {
		return htmlEl
	}
	return nil
}
