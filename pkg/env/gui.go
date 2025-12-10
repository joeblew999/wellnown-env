// gui.go: Via page registration functions for operations dashboards
//
// Provides reusable Via pages that services can register:
// - RegisterDashboardPage: Main dashboard with config, NATS, and dependencies
// - RegisterConfigPage: Detailed configuration view
//
// Services create their own Via instance and register the pages they need:
//
//	v := via.New()
//	v.Config(via.Options{ServerAddress: ":3000"})
//	env.RegisterDashboardPage(v, mgr, cfg)
//	env.RegisterConfigPage(v, mgr, cfg)
//	go v.Start()
package env

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
	"github.com/joeblew999/wellnown-env/pkg/env/registry"
)

// DashboardOptions configures the dashboard page
type DashboardOptions struct {
	// NavBar returns the navigation bar H element
	NavBar func(title string) h.H
}

// RegisterDashboardPage registers the main dashboard page (/) with Via
func RegisterDashboardPage(v *via.V, mgr *Manager, cfg interface{}, opts DashboardOptions) {
	fields := ExtractFields(mgr.Prefix(), cfg)

	v.Page("/", func(c *via.Context) {
		c.View(func() h.H {
			var navEl h.H
			if opts.NavBar != nil {
				navEl = opts.NavBar("Dashboard")
			}

			return h.Main(h.Class("container"),
				navEl,
				renderStatus(mgr),
				renderConfig(fields),
				renderDependencies(mgr, fields),
				renderNATS(mgr),
			)
		})
	})
}

// RegisterConfigPage registers the configuration detail page (/config) with Via
func RegisterConfigPage(v *via.V, mgr *Manager, cfg interface{}, opts DashboardOptions) {
	fields := ExtractFields(mgr.Prefix(), cfg)

	v.Page("/config", func(c *via.Context) {
		c.View(func() h.H {
			var navEl h.H
			if opts.NavBar != nil {
				navEl = opts.NavBar("Config")
			}

			return h.Main(h.Class("container"),
				navEl,
				h.H2(h.Text("Configuration")),
				renderConfigDetail(fields),
			)
		})
	})
}

// renderStatus renders the service status section
func renderStatus(mgr *Manager) h.H {
	reg := mgr.Registration()

	var statusItems []h.H

	// GitHub identity
	if reg != nil && reg.GitHub.Org != "" {
		statusItems = append(statusItems,
			h.Li(h.Strong(h.Text("Service: ")), h.Text(reg.GitHub.Name())),
		)
		if reg.GitHub.Tag != "" {
			statusItems = append(statusItems,
				h.Li(h.Strong(h.Text("Version: ")), h.Text(reg.GitHub.Tag)),
			)
		}
		if reg.GitHub.Commit != "" {
			statusItems = append(statusItems,
				h.Li(h.Strong(h.Text("Commit: ")), h.Text(reg.GitHub.Commit[:8])),
			)
		}
	}

	// Instance info
	if reg != nil {
		statusItems = append(statusItems,
			h.Li(h.Strong(h.Text("Instance: ")), h.Text(reg.Instance.ID)),
			h.Li(h.Strong(h.Text("Started: ")), h.Text(reg.Instance.Started.Format(time.RFC3339))),
		)
	}

	return h.Section(
		h.H2(h.Text("Status")),
		h.Ul(statusItems...),
	)
}

// renderConfig renders the configuration section
func renderConfig(fields []registry.FieldInfo) h.H {
	if len(fields) == 0 {
		return h.Section(
			h.H2(h.Text("Configuration")),
			h.P(h.Text("No configuration fields defined.")),
		)
	}

	var rows []h.H
	for _, f := range fields {
		if f.Dependency != "" {
			continue // Skip dependencies, shown separately
		}

		value := os.Getenv(f.EnvKey)
		if value == "" && f.Default != "" {
			value = f.Default + " (default)"
		}
		if f.IsSecret && value != "" {
			value = maskSecret(value)
		}

		required := ""
		if f.Required {
			required = " *"
		}

		rows = append(rows, h.Tr(
			h.Td(h.Text(f.Path+required)),
			h.Td(h.Code(h.Text(f.EnvKey))),
			h.Td(h.Text(value)),
		))
	}

	return h.Section(
		h.H2(h.Text("Configuration")),
		h.Table(h.Role("grid"),
			h.THead(
				h.Tr(
					h.Th(h.Text("Field")),
					h.Th(h.Text("Env Var")),
					h.Th(h.Text("Value")),
				),
			),
			h.TBody(rows...),
		),
	)
}

// renderConfigDetail renders the detailed configuration page
func renderConfigDetail(fields []registry.FieldInfo) h.H {
	if len(fields) == 0 {
		return h.P(h.Text("No configuration fields defined."))
	}

	var rows []h.H
	for _, f := range fields {
		value := os.Getenv(f.EnvKey)
		if value == "" && f.Default != "" {
			value = f.Default
		}
		if f.IsSecret && value != "" {
			value = maskSecret(value)
		}

		requiredText := "No"
		if f.Required {
			requiredText = "Yes"
		}

		secretText := "No"
		if f.IsSecret {
			secretText = "Yes"
		}

		depText := "-"
		if f.Dependency != "" {
			depText = f.Dependency
		}

		rows = append(rows, h.Tr(
			h.Td(h.Text(f.Path)),
			h.Td(h.Code(h.Text(f.Type))),
			h.Td(h.Code(h.Text(f.EnvKey))),
			h.Td(h.Text(f.Default)),
			h.Td(h.Text(requiredText)),
			h.Td(h.Text(secretText)),
			h.Td(h.Text(depText)),
			h.Td(h.Text(value)),
		))
	}

	return h.Table(
		h.THead(
			h.Tr(
				h.Th(h.Text("Field")),
				h.Th(h.Text("Type")),
				h.Th(h.Text("Env Var")),
				h.Th(h.Text("Default")),
				h.Th(h.Text("Required")),
				h.Th(h.Text("Secret")),
				h.Th(h.Text("Dependency")),
				h.Th(h.Text("Current Value")),
			),
		),
		h.TBody(rows...),
	)
}

// renderDependencies renders the service dependencies section
func renderDependencies(mgr *Manager, fields []registry.FieldInfo) h.H {
	deps := GetDependencies(fields)
	if len(deps) == 0 {
		return h.Div() // Empty if no dependencies
	}

	var items []h.H
	for _, dep := range deps {
		status := "unknown"
		if mgr.KV() != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			exists, err := ServiceExists(ctx, mgr.KV(), dep)
			cancel()
			if err == nil && exists {
				status = "available"
			} else {
				status = "unavailable"
			}
		}

		items = append(items, h.Li(
			h.Strong(h.Text(dep+": ")),
			h.Text(status),
		))
	}

	return h.Section(
		h.H2(h.Text("Dependencies")),
		h.Ul(items...),
	)
}

// renderNATS renders the NATS connection status section
func renderNATS(mgr *Manager) h.H {
	if mgr.natsNode == nil {
		return h.Section(
			h.H2(h.Text("NATS")),
			h.P(h.Text("NATS is disabled.")),
		)
	}

	items := []h.H{
		h.Li(h.Strong(h.Text("Client URL: ")), h.Code(h.Text(mgr.ClientURL()))),
		h.Li(h.Strong(h.Text("Node Name: ")), h.Text(mgr.natsNode.Name())),
	}

	if mgr.natsNode.IsLeaf() {
		items = append(items, h.Li(h.Strong(h.Text("Mode: ")), h.Text("Leaf (connected to hub)")))
	} else {
		items = append(items, h.Li(h.Strong(h.Text("Mode: ")), h.Text("Standalone")))
	}

	// Show registered services count
	if mgr.KV() != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		services, err := GetAllServices(ctx, mgr.KV())
		cancel()
		if err == nil {
			items = append(items, h.Li(h.Strong(h.Text("Registered Services: ")), h.Text(fmt.Sprintf("%d", len(services)))))
		}
	}

	return h.Section(
		h.H2(h.Text("NATS")),
		h.Ul(items...),
	)
}

// maskSecret masks a secret value for display
func maskSecret(value string) string {
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}
