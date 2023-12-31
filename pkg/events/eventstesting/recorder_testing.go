package eventstesting

import (
	"context"
	"fmt"
	"testing"

	"github.com/mfojtik/controller-framework/pkg/events"
)

type TestingEventRecorder struct {
	t         *testing.T
	component string
}

func (r *TestingEventRecorder) WithContext(ctx context.Context) events.Recorder {
	return r
}

// NewTestingEventRecorder provides event recorder that will log all recorded events to the error log.
func NewTestingEventRecorder(t *testing.T) events.Recorder {
	return &TestingEventRecorder{t: t, component: "test"}
}

func (r *TestingEventRecorder) ComponentName() string {
	return r.component
}

func (r *TestingEventRecorder) ForComponent(c string) events.Recorder {
	return &TestingEventRecorder{t: r.t, component: c}
}

func (r *TestingEventRecorder) Shutdown() {}

func (r *TestingEventRecorder) WithComponentSuffix(suffix string) events.Recorder {
	return r.ForComponent(fmt.Sprintf("%s-%s", r.ComponentName(), suffix))
}

func (r *TestingEventRecorder) Event(reason, message string) {
	r.t.Logf("Event: %v: %v", reason, message)
}

func (r *TestingEventRecorder) Eventf(reason, messageFmt string, args ...interface{}) {
	r.Event(reason, fmt.Sprintf(messageFmt, args...))
}

func (r *TestingEventRecorder) Warning(reason, message string) {
	r.t.Logf("Warning: %v: %v", reason, message)
}

func (r *TestingEventRecorder) Warningf(reason, messageFmt string, args ...interface{}) {
	r.Warning(reason, fmt.Sprintf(messageFmt, args...))
}
