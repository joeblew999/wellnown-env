# GitHub Issue for go-via/via

**Title:** Add Handler() method for testing integration with gost-dom

**Labels:** enhancement

---

## Summary

When testing Via applications with headless browser frameworks like [gost-dom](https://github.com/gost-dom/browser), there's no way to access Via's internal HTTP handler. This makes it impossible to use `browser.WithHandler()` for direct handler testing without TCP overhead.

## Problem

Currently, Via only exposes `Start()` which binds to a port. For testing, we need to pass the handler directly to test frameworks:

```go
// This doesn't work - no way to get the handler
b := browser.New(
    browser.WithHandler(v.Handler()), // Handler() doesn't exist
)
```

The workaround is to start a real server and use HTTP, which adds:
- Port management complexity
- TCP overhead (~2-5s vs ~50ms)
- Potential port conflicts in parallel tests

## Proposed Solution

Add a `Handler()` method to expose the internal mux:

```go
// Handler returns the underlying http.Handler for use with custom servers or testing.
func (v *V) Handler() http.Handler {
    return v.mux
}
```

This is a one-line addition that enables:

1. **Direct handler testing** with gost-dom, httptest, etc.
2. **Custom server integration** (embedding Via in existing servers)
3. **Middleware wrapping** for auth, logging, etc.

## Additional Fix: SSE Closed Pipe Errors

During testing, when browsers close before SSE handlers detect the closure, Via logs errors like:
```
[error] msg="PatchElements failed: failed to write to response writer: io: read/write on closed pipe"
```

This is noise during normal shutdown. Suggest checking context before logging:

```go
if err := sse.PatchElements(patch.content); err != nil {
    // Only log if connection wasn't closed
    if sse.Context().Err() == nil {
        v.logErr(c, "PatchElements failed: %v", err)
    }
    continue
}
```

## Reference Implementation

I have a working fork with both changes:
- Repository: https://github.com/joeblew999/via
- Branch: `feature/handler-method`
- Commits:
  - `9cc0174` - Add Handler() method
  - `fa2c5c0` - Suppress closed pipe errors

## Test Results

With these changes, Via + gost-dom integration tests work well:
- Full SSE/Datastar reactivity testing
- Button clicks triggering server actions
- ~50ms test execution (vs seconds with real browser)

Happy to submit a PR if you're interested in these additions.

---

## To submit this issue:

1. Go to: https://github.com/go-via/via/issues/new
2. Copy the content above (everything between the `---` lines)
3. Paste and submit
