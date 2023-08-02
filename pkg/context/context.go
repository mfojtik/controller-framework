package context

import (
	"fmt"
	"github.com/mfojtik/controller-framework/pkg/events"
	"github.com/mfojtik/controller-framework/pkg/framework"
	"k8s.io/apimachinery/pkg/runtime"
	runtime2 "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"strings"
)

// Context implements Context and provide user access to queue and object that caused
// the sync to be triggered.
type Context struct {
	eventRecorder events.Recorder
	queue         workqueue.RateLimitingInterface
	queueKey      string
	name          string
}

var _ framework.Context = Context{}

// New gives new sync context.
func New(name string, recorder events.Recorder) framework.Context {
	return Context{
		queue: workqueue.NewRateLimitingQueueWithConfig(workqueue.DefaultControllerRateLimiter(), workqueue.RateLimitingQueueConfig{
			Name: name,
		}),
		name:          name,
		eventRecorder: recorder.WithComponentSuffix(strings.ToLower(name)),
	}
}

func NewWithQueueKey(ctx *Context, keyName string) {
	ctx.queueKey = keyName
}

func (c Context) Queue() workqueue.RateLimitingInterface {
	return c.queue
}

func (c Context) WithQueueKey(key string) framework.Context {
	c.queueKey = key
	return c
}

func (c Context) QueueKey() string {
	return c.queueKey
}

func (c Context) Recorder() events.Recorder {
	return c.eventRecorder
}

// EventHandler provides default event handler that is added to an informers passed to controller factory.
func (c Context) EventHandler(queueKeysFunc framework.ObjectQueueKeysFunc, filter framework.EventFilterFunc) cache.ResourceEventHandler {
	resourceEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			runtimeObj, ok := obj.(runtime.Object)
			if !ok {
				runtime2.HandleError(fmt.Errorf("added object %+v is not runtime Object", obj))
				return
			}
			c.enqueueKeys(queueKeysFunc(runtimeObj)...)
		},
		UpdateFunc: func(old, new interface{}) {
			runtimeObj, ok := new.(runtime.Object)
			if !ok {
				runtime2.HandleError(fmt.Errorf("updated object %+v is not runtime Object", runtimeObj))
				return
			}
			c.enqueueKeys(queueKeysFunc(runtimeObj)...)
		},
		DeleteFunc: func(obj interface{}) {
			runtimeObj, ok := obj.(runtime.Object)
			if !ok {
				if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
					c.enqueueKeys(queueKeysFunc(tombstone.Obj.(runtime.Object))...)

					return
				}
				runtime2.HandleError(fmt.Errorf("updated object %+v is not runtime Object", runtimeObj))
				return
			}
			c.enqueueKeys(queueKeysFunc(runtimeObj)...)
		},
	}
	if filter == nil {
		return resourceEventHandler
	}
	return cache.FilteringResourceEventHandler{
		FilterFunc: filter,
		Handler:    resourceEventHandler,
	}
}

func (c Context) enqueueKeys(keys ...string) {
	for _, qKey := range keys {
		c.queue.Add(qKey)
	}
}
