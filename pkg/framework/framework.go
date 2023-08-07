package framework

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"

	"github.com/mfojtik/controller-framework/pkg/events"
)

// Controller interface represents a runnable Kubernetes controller.
// Cancelling the syncContext passed will cause the controller to shutdown.
// Number of workers determine how much parallel the job processing should be.
type Controller interface {
	// Run runs the controller and blocks until the controller is finished.
	// Number of workers can be specified via workers parameter.
	// This function will return when all internal loops are finished.
	// Note that having more than one worker usually means handing parallelization of Sync().
	Run(ctx context.Context, workers int)

	// Sync contain the main controller logic.
	// This should not be called directly, but can be used in unit tests to exercise the sync.
	Sync(ctx context.Context, controllerContext Context) error

	// Name returns the controller name string.
	Name() string
}

// Context interface represents a context given to the Sync() function where the main controller logic happen.
// Context exposes controller name and give user access to the queue (for manual requeue).
// Context also provides metadata about object that informers observed as changed.
type Context interface {
	// Queue gives access to controller queue. This can be used for manual requeue, although if a Sync() function return
	// an error, the object is automatically re-queued. Use with caution.
	Queue() workqueue.RateLimitingInterface

	// QueueKey represents the queue key passed to the Sync function.
	QueueKey() string

	// WithQueueKey takes a string key and return a the same context with the new key set.
	// This is used when passing the context to sync function in case the key is different than the default key.
	// This should be used only when you want to run sync() with different key.
	WithQueueKey(key string) Context

	// Recorder provide access to event recorder.
	Recorder() events.Recorder
}

type ControllerSyncPanicFn func(interface{}) error
type ControllerSyncErrorFn func(error) error

// ControllerSyncFn is a function that contain main controller logic.
// The syncContext.syncContext passed is the main controller syncContext, when cancelled it means the controller is being shut down.
// The syncContext provides access to controller name, queue and event recorder.
type ControllerSyncFn func(ctx context.Context, controllerContext Context) error

// PostStartHook specify a function that will run after controller is started.
// The context is cancelled when the controller is asked to shutdown and the post start hook should terminate as well.
// The syncContext allow access to controller queue and event recorder.
type PostStartHook func(ctx context.Context, syncContext Context) error

// ObjectQueueKeysFunc is used to make a string work queue keys out of the runtime object that is passed to it.
// This can extract the "namespace/name" if you need to or just return "key" if you building controller that only use string
// triggers.
type ObjectQueueKeysFunc func(runtime.Object) []string

// EventFilterFunc is used to filter informer events to prevent Sync() from being called
type EventFilterFunc func(obj interface{}) bool

// DefaultQueueKey is the queue key used for string trigger based controllers.
const DefaultQueueKey = "key"
