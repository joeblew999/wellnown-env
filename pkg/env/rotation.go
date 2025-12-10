// rotation.go: Secret rotation notifications
//
// Allows services to subscribe to secret rotation events.
// When secrets are rotated (via external rotation service or cloud),
// services receive notifications and can gracefully restart.
//
// Subject pattern: secrets.rotated.{secret_path}
package env

import (
	"github.com/nats-io/nats.go"
)

const rotationSubjectPrefix = "secrets.rotated."

// RotationHandler handles secret rotation events
type RotationHandler func(path string)

// OnRotate subscribes to secret rotation notifications
// The handler is called with the secret path when a rotation occurs
func OnRotate(nc *nats.Conn, handler RotationHandler) (*nats.Subscription, error) {
	return nc.Subscribe(rotationSubjectPrefix+">", func(msg *nats.Msg) {
		// Extract path from subject
		path := msg.Subject[len(rotationSubjectPrefix):]
		handler(path)
	})
}

// PublishRotation publishes a secret rotation event
// This is typically called by a rotation service, not by normal services
func PublishRotation(nc *nats.Conn, path string) error {
	return nc.Publish(rotationSubjectPrefix+path, nil)
}
