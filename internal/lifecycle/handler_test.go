// Package lifecycle_test tests lifecycle handler integration with notification and history systems.
// Related: /home/ari/repos/autospec/internal/lifecycle/lifecycle.go
// Tags: lifecycle, handler, integration, notification, history

package lifecycle_test

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
)

// TestNotifyHandlerSatisfiesInterface verifies that *notify.Handler
// satisfies the lifecycle.NotificationHandler interface.
// This compile-time check ensures the interface is correctly defined.
func TestNotifyHandlerSatisfiesInterface(t *testing.T) {
	t.Parallel()

	// This assignment will fail to compile if notify.Handler
	// doesn't satisfy lifecycle.NotificationHandler
	var _ lifecycle.NotificationHandler = (*notify.Handler)(nil)
}
