package factory

import (
	"fmt"
	"github.com/mfojtik/controller-framework/pkg/context"
	"github.com/mfojtik/controller-framework/pkg/controller"
	"github.com/mfojtik/controller-framework/pkg/framework"
	corev1 "k8s.io/api/core/v1"
	"time"

	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/runtime"
	errorutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	"github.com/mfojtik/controller-framework/pkg/events"
	//operatorv1helpers "github.com/mfojtik/controller-framework/pkg/operator/v1helpers"
)

// DefaultQueueKeysFunc returns a slice with a single element - the DefaultQueueKey
func DefaultQueueKeysFunc(_ runtime.Object) []string {
	return []string{framework.DefaultQueueKey}
}

var defaultCacheSyncTimeout = 10 * time.Minute

// Factory is generator that generate standard Kubernetes controllers.
// Factory is really generic and should be only used for simple controllers that does not require special stuff..
type Factory struct {
	sync        framework.ControllerSyncFn
	syncContext framework.Context

	//syncDegradedClient    operatorv1helpers.OperatorClient
	resyncInterval  time.Duration
	resyncSchedules []string

	informers          []filteredInformers
	informerQueueKeys  []informersWithQueueKey
	bareInformers      []framework.Informer
	namespaceInformers []*namespaceInformer
	cachesToSync       []cache.InformerSynced

	postStartHooks        []framework.PostStartHook
	interestingNamespaces sets.Set[string]

	controllerPanicHandler framework.ControllerSyncPanicFn
	controllerErrorHandler framework.ControllerSyncErrorFn
}

type namespaceInformer struct {
	informer framework.Informer
	nsFilter framework.EventFilterFunc
}

type informersWithQueueKey struct {
	informers  []framework.Informer
	filter     framework.EventFilterFunc
	queueKeyFn framework.ObjectQueueKeysFunc
}

type filteredInformers struct {
	informers []framework.Informer
	filter    framework.EventFilterFunc
}

// ObjectQueueKeyFunc is used to make a string work queue key out of the runtime object that is passed to it.
// This can extract the "namespace/name" if you need to or just return "key" if you building controller that only use string
// triggers.
// DEPRECATED: use ObjectQueueKeysFunc instead
type ObjectQueueKeyFunc func(runtime.Object) string

// New return new factory instance.
func New() *Factory {
	return &Factory{}
}

// WithSync is used to set the controller synchronization function. This function is the core of the controller and is
// usually hold the main controller logic.
func (f *Factory) WithSync(syncFn framework.ControllerSyncFn) *Factory {
	f.sync = syncFn
	return f
}

// WithInformers is used to register event handlers and get the caches synchronized functions.
// Pass informers you want to use to react to changes on resources. If informer event is observed, then the Sync() function
// is called.
func (f *Factory) WithInformers(informers ...framework.Informer) *Factory {
	f.WithFilteredEventsInformers(nil, informers...)
	return f
}

// WithFilteredEventsInformers is used to register event handlers and get the caches synchronized functions.
// Pass the informers you want to use to react to changes on resources. If informer event is observed, then the Sync() function
// is called.
// Pass filter to filter out events that should not trigger Sync() call.
func (f *Factory) WithFilteredEventsInformers(filter framework.EventFilterFunc, informers ...framework.Informer) *Factory {
	f.informers = append(f.informers, filteredInformers{
		informers: informers,
		filter:    filter,
	})
	return f
}

// WithBareInformers allow to register informer that already has custom event handlers registered and no additional
// event handlers will be added to this informer.
// The controller will wait for the cache of this informer to be synced.
// The existing event handlers will have to respect the queue key function or the sync() implementation will have to
// count with custom queue keys.
func (f *Factory) WithBareInformers(informers ...framework.Informer) *Factory {
	f.bareInformers = append(f.bareInformers, informers...)
	return f
}

// WithInformersQueueKeyFunc is used to register event handlers and get the caches synchronized functions.
// Pass informers you want to use to react to changes on resources. If informer event is observed, then the Sync() function
// is called.
// Pass the queueKeyFn you want to use to transform the informer runtime.Object into string key used by work queue.
func (f *Factory) WithInformersQueueKeyFunc(queueKeyFn ObjectQueueKeyFunc, informers ...framework.Informer) *Factory {
	f.informerQueueKeys = append(f.informerQueueKeys, informersWithQueueKey{
		informers: informers,
		queueKeyFn: func(o runtime.Object) []string {
			return []string{queueKeyFn(o)}
		},
	})
	return f
}

// WithFilteredEventsInformersQueueKeyFunc is used to register event handlers and get the caches synchronized functions.
// Pass informers you want to use to react to changes on resources. If informer event is observed, then the Sync() function
// is called.
// Pass the queueKeyFn you want to use to transform the informer runtime.Object into string key used by work queue.
// Pass filter to filter out events that should not trigger Sync() call.
func (f *Factory) WithFilteredEventsInformersQueueKeyFunc(queueKeyFn ObjectQueueKeyFunc, filter framework.EventFilterFunc, informers ...framework.Informer) *Factory {
	f.informerQueueKeys = append(f.informerQueueKeys, informersWithQueueKey{
		informers: informers,
		filter:    filter,
		queueKeyFn: func(o runtime.Object) []string {
			return []string{queueKeyFn(o)}
		},
	})
	return f
}

// WithInformersQueueKeysFunc is used to register event handlers and get the caches synchronized functions.
// Pass informers you want to use to react to changes on resources. If informer event is observed, then the Sync() function
// is called.
// Pass the queueKeyFn you want to use to transform the informer runtime.Object into string key used by work queue.
func (f *Factory) WithInformersQueueKeysFunc(queueKeyFn framework.ObjectQueueKeysFunc, informers ...framework.Informer) *Factory {
	f.informerQueueKeys = append(f.informerQueueKeys, informersWithQueueKey{
		informers:  informers,
		queueKeyFn: queueKeyFn,
	})
	return f
}

// WithFilteredEventsInformersQueueKeysFunc is used to register event handlers and get the caches synchronized functions.
// Pass informers you want to use to react to changes on resources. If informer event is observed, then the Sync() function
// is called.
// Pass the queueKeyFn you want to use to transform the informer runtime.Object into string key used by work queue.
// Pass filter to filter out events that should not trigger Sync() call.
func (f *Factory) WithFilteredEventsInformersQueueKeysFunc(queueKeyFn framework.ObjectQueueKeysFunc, filter framework.EventFilterFunc, informers ...framework.Informer) *Factory {
	f.informerQueueKeys = append(f.informerQueueKeys, informersWithQueueKey{
		informers:  informers,
		filter:     filter,
		queueKeyFn: queueKeyFn,
	})
	return f
}

// WithPostStartHooks allows to register functions that will run asynchronously after the controller is started via Run command.
func (f *Factory) WithPostStartHooks(hooks ...framework.PostStartHook) *Factory {
	f.postStartHooks = append(f.postStartHooks, hooks...)
	return f
}

// WithNamespaceInformer is used to register event handlers and get the caches synchronized functions.
// The sync function will only trigger when the object observed by this informer is a namespace and its name matches the interestingNamespaces.
// Do not use this to register non-namespace informers.
func (f *Factory) WithNamespaceInformer(informer framework.Informer, interestingNamespaces ...string) *Factory {
	f.namespaceInformers = append(f.namespaceInformers, &namespaceInformer{
		informer: informer,
		nsFilter: namespaceChecker(interestingNamespaces),
	})
	return f
}

// namespaceChecker returns a function which returns true if an inpuut obj
// (or its tombstone) is a namespace  and it matches a name of any namespaces
// that we are interested in
func namespaceChecker(interestingNamespaces []string) func(obj interface{}) bool {
	interestingNamespacesSet := sets.NewString(interestingNamespaces...)

	return func(obj interface{}) bool {
		ns, ok := obj.(*corev1.Namespace)
		if ok {
			return interestingNamespacesSet.Has(ns.Name)
		}

		// the object might be getting deleted
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if ok {
			if ns, ok := tombstone.Obj.(*corev1.Namespace); ok {
				return interestingNamespacesSet.Has(ns.Name)
			}
		}
		return false
	}
}

// ResyncEvery will cause the Sync() function to be called periodically, regardless of informers.
// This is useful when you want to refresh every N minutes or you fear that your informers can be stucked.
// If this is not called, no periodical resync will happen.
// Note: The controller context passed to Sync() function in this case does not contain the object metadata or object itself.
//
//	This can be used to detect periodical resyncs, but normal Sync() have to be cautious about `nil` objects.
func (f *Factory) ResyncEvery(interval time.Duration) *Factory {
	f.resyncInterval = interval
	return f
}

// ResyncSchedule allows to supply a Cron syntax schedule that will be used to schedule the sync() call runs.
// This allows more fine-tuned controller scheduling than ResyncEvery.
// Examples:
//
// factory.New().ResyncSchedule("@every 1s").ToController()     // Every second
// factory.New().ResyncSchedule("@hourly").ToController()       // Every hour
// factory.New().ResyncSchedule("30 * * * *").ToController()	// Every hour on the half hour
//
// Note: The controller context passed to Sync() function in this case does not contain the object metadata or object itself.
//
//	This can be used to detect periodical resyncs, but normal Sync() have to be cautious about `nil` objects.
func (f *Factory) ResyncSchedule(schedules ...string) *Factory {
	f.resyncSchedules = append(f.resyncSchedules, schedules...)
	return f
}

// WithSyncContext allows to specify custom, existing sync context for this factory.
// This is useful during unit testing where you can override the default event recorder or mock the runtime objects.
// If this function not called, a Context is created by the factory automatically.
func (f *Factory) WithSyncContext(ctx framework.Context) *Factory {
	f.syncContext = ctx
	return f
}

// WithSyncErrorHandler allows in case the sync() function return error to additionally handle the error.
// This allows to build error handling mechanisms that for example report operator status or provide error count metrics for controller.
// NOTE: The original error is always returned from sync() call (so it is retried as usual).
// NOTE2: If the error is SyntheticRequeueError this error is not being handled and the sync() is simply retried.
// NOTE3: If an error is returned from the handler, this error causes panic()
func (f *Factory) WithSyncErrorHandler(fn func(err error) error) *Factory {
	f.controllerErrorHandler = fn
	return f
}

// WithSyncPanicHandler allows to register a panic() handler for Run() method of controller. This handler will recover from
// any panic inside the sync() and pass the panic into given function. This allows to update operator status OR it allows specific
// controller metrics to be created.
// If the panic handler is not specified, the util.HandleCrash() is called as usual.
func (f *Factory) WithSyncPanicHandler(fn func(panicInterface interface{}) error) *Factory {
	f.controllerPanicHandler = fn
	return f
}

// WithSyncDegradedOnError encapsulate the controller sync() function, so when this function return an error, the operator client
// is used to set the degraded condition to (eg. "ControllerFooDegraded"). The degraded condition name is set based on the controller name.
/*
func (f *Factory) WithSyncDegradedOnError(operatorClient operatorv1helpers.OperatorClient) *Factory {
	f.syncDegradedClient = operatorClient
	return f
}
*/

// Controller produce a runnable controller.
func (f *Factory) ToController(name string, eventRecorder events.Recorder) framework.Controller {
	if f.sync == nil {
		panic(fmt.Errorf("WithSync() must be used before calling ToController() in %q", name))
	}

	var ctx framework.Context
	if f.syncContext != nil {
		ctx = f.syncContext
	} else {
		ctx = context.New(name, eventRecorder)
	}

	var cronSchedules []cron.Schedule
	if len(f.resyncSchedules) > 0 {
		var errors []error
		for _, schedule := range f.resyncSchedules {
			if s, err := cron.ParseStandard(schedule); err != nil {
				errors = append(errors, err)
			} else {
				cronSchedules = append(cronSchedules, s)
			}
		}
		if err := errorutil.NewAggregate(errors); err != nil {
			panic(fmt.Errorf("failed to parse controller schedules for %q: %v", name, err))
		}
	}

	informersToSync := []cache.InformerSynced{}

	for i := range f.informerQueueKeys {
		for d := range f.informerQueueKeys[i].informers {
			informer := f.informerQueueKeys[i].informers[d]
			queueKeyFn := f.informerQueueKeys[i].queueKeyFn
			if _, err := informer.AddEventHandler(ctx.(context.Context).EventHandler(queueKeyFn, f.informerQueueKeys[i].filter)); err != nil {
				panic(err)
			}
			informersToSync = append(informersToSync, informer.HasSynced)
		}
	}

	for i := range f.informers {
		for d := range f.informers[i].informers {
			informer := f.informers[i].informers[d]
			if _, err := informer.AddEventHandler(ctx.(context.Context).EventHandler(DefaultQueueKeysFunc, f.informers[i].filter)); err != nil {
				panic(err)
			}
			informersToSync = append(informersToSync, informer.HasSynced)
		}
	}

	for i := range f.bareInformers {
		informersToSync = append(informersToSync, f.bareInformers[i].HasSynced)
	}

	for i := range f.namespaceInformers {
		if _, err := f.namespaceInformers[i].informer.AddEventHandler(ctx.(context.Context).EventHandler(DefaultQueueKeysFunc, f.namespaceInformers[i].nsFilter)); err != nil {
			panic(err)
		}
		informersToSync = append(informersToSync, f.namespaceInformers[i].informer.HasSynced)
	}

	f.cachesToSync = append(f.cachesToSync, informersToSync...)

	c := controller.New(
		name,
		f.sync,
		ctx,
		f.resyncInterval,
		cronSchedules,
		f.postStartHooks,
		append([]cache.InformerSynced{}, f.cachesToSync...),
		defaultCacheSyncTimeout,
	)

	return c
}
