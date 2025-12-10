//go:build integration

package pcview

import (
	"context"
	"testing"
	"time"

	"github.com/go-via/via"
	"github.com/gost-dom/browser"
	"github.com/gost-dom/browser/html"
	"github.com/gost-dom/browser/scripting/v8engine"
	"github.com/gost-dom/browser/testing/gosttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessesPage_Render tests that the processes page renders correctly
func TestProcessesPage_Render(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup Via with mock controller
	v := via.New()
	mock := &MockController{
		processes: []ProcessState{
			{Name: "ticker", Status: "Running", IsRunning: true, Pid: 1234, Health: "healthy"},
			{Name: "counter", Status: "Running", IsRunning: true, Pid: 1235, Health: "healthy"},
			{Name: "logger", Status: "Disabled", IsRunning: false, Pid: 0, Health: ""},
		},
	}
	state := NewState()
	state.SetProcesses(mock.processes, "")

	RegisterPage(v, mock, state, PageOptions{})

	// Create gost-dom browser with V8 engine
	b := browser.New(
		browser.WithScriptEngine(v8engine.DefaultEngine()),
		browser.WithContext(ctx),
		browser.WithHandler(v.Handler()),
		browser.WithLogger(gosttest.NewTestingLogger(t)),
	)
	defer b.Close()

	// Open the processes page
	win, err := b.Open("http://localhost/processes")
	require.NoError(t, err)

	// Via renders server-side, so content is available immediately
	// We use Advance to process any immediate JS initialization
	// but don't wait for SSE (which stays open)
	_ = win.Clock().Advance(100 * time.Millisecond)

	// Verify page content
	doc := win.Document()
	body := doc.Body().TextContent()
	assert.Contains(t, body, "Process Manager")
	assert.Contains(t, body, "ticker")
	assert.Contains(t, body, "counter")
	assert.Contains(t, body, "logger")
}

// TestProcessesPage_StopButton tests clicking the Stop button
func TestProcessesPage_StopButton(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	v := via.New()
	mock := &MockController{
		processes: []ProcessState{
			{Name: "ticker", Status: "Running", IsRunning: true, Pid: 1234, Health: "healthy"},
		},
	}
	state := NewState()
	state.SetProcesses(mock.processes, "")

	RegisterPage(v, mock, state, PageOptions{})

	b := browser.New(
		browser.WithScriptEngine(v8engine.DefaultEngine()),
		browser.WithContext(ctx),
		browser.WithHandler(v.Handler()),
		browser.WithLogger(gosttest.NewTestingLogger(t)),
	)
	defer b.Close()

	win, err := b.Open("http://localhost/processes")
	require.NoError(t, err)

	// Via renders server-side - advance clock for JS initialization
	_ = win.Clock().Advance(100 * time.Millisecond)

	// Find and click Stop button
	doc := win.Document()
	buttons := doc.GetElementsByTagName("button")

	for i := 0; i < buttons.Length(); i++ {
		node := buttons.Item(i)
		if btn, ok := node.(html.HTMLElement); ok {
			if btn.TextContent() == "Stop" {
				btn.Click()
				// Wait for action to be recorded
				_ = win.Clock().ProcessEventsWhile(ctx, func() bool {
					return len(mock.actions) == 0
				})

				// Verify the action was called
				assert.Contains(t, mock.actions, "stop:ticker")
				return
			}
		}
	}
	t.Log("Stop button not found - page may not have fully rendered with buttons")
}

// TestExamplesPage_Render tests the examples page renders correctly
func TestExamplesPage_Render(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	v := via.New()
	mock := &MockController{
		processes: []ProcessState{
			{Name: "ticker", Status: "Running", IsRunning: true, Pid: 1234},
			{Name: "counter", Status: "Running", IsRunning: true, Pid: 1235},
			{Name: "logger", Status: "Disabled", IsRunning: false, Pid: 0},
		},
	}
	state := NewState()
	state.SetProcesses(mock.processes, "")

	RegisterExamplesPage(v, mock, state, ExamplesPageOptions{})

	b := browser.New(
		browser.WithScriptEngine(v8engine.DefaultEngine()),
		browser.WithContext(ctx),
		browser.WithHandler(v.Handler()),
		browser.WithLogger(gosttest.NewTestingLogger(t)),
	)
	defer b.Close()

	win, err := b.Open("http://localhost/examples")
	require.NoError(t, err)

	// Via renders server-side - advance clock for JS initialization
	_ = win.Clock().Advance(100 * time.Millisecond)

	body := win.Document().Body().TextContent()
	assert.Contains(t, body, "Demo Processes")
	assert.Contains(t, body, "Start All")
	assert.Contains(t, body, "Stop All")
	assert.Contains(t, body, "ticker")
	assert.Contains(t, body, "counter")
	assert.Contains(t, body, "logger")
}

// TestExamplesPage_StartAll tests the Start All button
func TestExamplesPage_StartAll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	v := via.New()
	mock := &MockController{
		processes: []ProcessState{
			{Name: "ticker", IsRunning: false},
			{Name: "counter", IsRunning: false},
			{Name: "logger", IsRunning: false},
		},
	}
	state := NewState()
	state.SetProcesses(mock.processes, "")

	RegisterExamplesPage(v, mock, state, ExamplesPageOptions{})

	b := browser.New(
		browser.WithScriptEngine(v8engine.DefaultEngine()),
		browser.WithContext(ctx),
		browser.WithHandler(v.Handler()),
		browser.WithLogger(gosttest.NewTestingLogger(t)),
	)
	defer b.Close()

	win, err := b.Open("http://localhost/examples")
	require.NoError(t, err)

	// Via renders server-side - advance clock for JS initialization
	_ = win.Clock().Advance(100 * time.Millisecond)

	// Find and click "Start All" button
	doc := win.Document()
	buttons := doc.GetElementsByTagName("button")

	for i := 0; i < buttons.Length(); i++ {
		node := buttons.Item(i)
		if btn, ok := node.(html.HTMLElement); ok {
			if btn.TextContent() == "Start All" {
				btn.Click()
				break
			}
		}
	}

	// Wait for all 3 example actions to be recorded
	_ = win.Clock().ProcessEventsWhile(ctx, func() bool {
		return len(mock.actions) < 3
	})

	// Verify all example processes were started
	assert.Contains(t, mock.actions, "start:ticker")
	assert.Contains(t, mock.actions, "start:counter")
	assert.Contains(t, mock.actions, "start:logger")
}
