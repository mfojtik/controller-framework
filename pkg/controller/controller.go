package controller

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	//"github.com/mfojtik/controller-framework/pkg/operator/management"
	//operatorv1helpers "github.com/mfojtik/controller-framework/pkg/operator/v1helpers"
	//operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/robfig/cron"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/mfojtik/controller-framework/pkg/framework"
)

// SyntheticRequeueError can be returned from sync() in case of forcing a sync() retry artificially.
// This can be also done by re-adding the key to queue, but this is cheaper and more convenient.
var SyntheticRequeueError = errors.New("synthetic requeue request")

// baseController represents generic Kubernetes controller boiler-plate
type baseController struct {
	name        string
	sync        func(ctx context.Context, controllerContext framework.Context) error
	syncContext framework.Context

	resyncEvery     time.Duration
	resyncSchedules []cron.Schedule

	postStartHooks []framework.PostStartHook

	informerSynced        []cache.InformerSynced
	informerSyncedTimeout time.Duration

	syncPanicHandler framework.ControllerSyncPanicFn
	syncErrorHandler framework.ControllerSyncErrorFn
}

func New(
	name string,
	sync func(ctx context.Context, controllerContext framework.Context) error,
	syncContext framework.Context,
	resyncEvery time.Duration,
	resyncSchedules []cron.Schedule,
	postStartHooks []framework.PostStartHook,
	cachesToSync []cache.InformerSynced,
	cacheSyncTimeout time.Duration) framework.Controller {
	return &baseController{name: name, informerSynced: cachesToSync, sync: sync, syncContext: syncContext, resyncEvery: resyncEvery, resyncSchedules: resyncSchedules, postStartHooks: postStartHooks, informerSyncedTimeout: cacheSyncTimeout}
}

var _ framework.Controller = &baseController{}

func (c *baseController) Name() string {
	return c.name
}

type scheduledJob struct {
	queue workqueue.RateLimitingInterface
	name  string
}

func newScheduledJob(name string, queue workqueue.RateLimitingInterface) cron.Job {
	return &scheduledJob{
		queue: queue,
		name:  name,
	}
}

func (s *scheduledJob) Run() {
	klog.V(4).Infof("Triggering scheduled %q controller run", s.name)
	s.queue.Add(framework.DefaultQueueKey)
}

func waitForNamedCacheSync(controllerName string, stopCh <-chan struct{}, cacheSyncs ...cache.InformerSynced) error {
	if len(cacheSyncs) == 0 {
		return nil
	}
	klog.Infof("Waiting for informer caches to sync for %s controller ...", controllerName)

	if !cache.WaitForCacheSync(stopCh, cacheSyncs...) {
		return fmt.Errorf("unable to sync caches for %s", controllerName)
	}

	klog.Infof("Informer caches for %s controller are synced", controllerName)

	return nil
}

func (c *baseController) Run(ctx context.Context, workers int) {
	// HandleCrash recovers panics
	defer utilruntime.HandleCrash(func(in interface{}) {
		if c.syncPanicHandler == nil {
			panic(in)
		}
		if err := c.syncPanicHandler(in); err != nil {
			klog.Warningf("PANIC: Detected panic() in controller %q failed to run panic handler: %v\n\n%s", c.Name(), err, in)
		}
	})

	// give caches 10 minutes to sync
	cacheSyncCtx, cacheSyncCancel := context.WithTimeout(ctx, c.informerSyncedTimeout)
	defer cacheSyncCancel()
	err := waitForNamedCacheSync(c.name, cacheSyncCtx.Done(), c.informerSynced...)
	if err != nil {
		select {
		case <-ctx.Done():
			// Exit gracefully because the controller was requested to stop.
			return
		default:
			// If caches did not sync after 10 minutes, it has taken oddly long and
			// we should provide feedback. Since the control loops will never start,
			// it is safer to exit with a good message than to continue with a dead loop.
			// TODO: Consider making this behavior configurable.
			klog.Exit(err)
		}
	}

	var workerWg sync.WaitGroup
	defer func() {
		defer klog.Infof("All %s workers have been terminated", c.name)
		workerWg.Wait()
	}()

	// queueContext is used to track and initiate queue shutdown
	queueContext, queueContextCancel := context.WithCancel(context.TODO())

	for i := 1; i <= workers; i++ {
		klog.Infof("Starting worker #%d for controller %s  ...", i, c.name)
		workerWg.Add(1)
		go func() {
			defer func() {
				klog.Infof("Shutting down worker of %s controller ...", c.name)
				workerWg.Done()
			}()
			c.runWorker(queueContext)
		}()
	}

	// if scheduled run is requested, run the cron scheduler
	if c.resyncSchedules != nil {
		scheduler := cron.New()
		for _, s := range c.resyncSchedules {
			scheduler.Schedule(s, newScheduledJob(c.name, c.syncContext.Queue()))
		}
		scheduler.Start()
		defer scheduler.Stop()
	}

	// runPeriodicalResync is independent from queue
	if c.resyncEvery > 0 {
		workerWg.Add(1)
		if c.resyncEvery < 60*time.Second {
			// Warn about too fast resyncs as they might drain the operators QPS.
			// This event is cheap as it is only emitted on operator startup.
			c.syncContext.Recorder().Warningf("FastControllerResync", "Controller %q resync interval is set to %s which might lead to client request throttling", c.name, c.resyncEvery)
		}
		go func() {
			defer workerWg.Done()
			wait.UntilWithContext(ctx, func(ctx context.Context) { c.syncContext.Queue().Add(framework.DefaultQueueKey) }, c.resyncEvery)
		}()
	}

	// run post-start hooks (custom triggers, etc.)
	if len(c.postStartHooks) > 0 {
		var hookWg sync.WaitGroup
		defer func() {
			hookWg.Wait() // wait for the post-start hooks
			klog.Infof("All %s post start hooks have been terminated", c.name)
		}()
		for i := range c.postStartHooks {
			hookWg.Add(1)
			go func(index int) {
				defer hookWg.Done()
				if err := c.postStartHooks[index](ctx, c.syncContext); err != nil {
					klog.Warningf("%s controller post start hook error: %v", c.name, err)
				}
			}(i)
		}
	}

	// Handle controller shutdown

	<-ctx.Done()                     // wait for controller context to be cancelled
	c.syncContext.Queue().ShutDown() // shutdown the controller queue first
	queueContextCancel()             // cancel the queue context, which tell workers to initiate shutdown

	// Wait for all workers to finish their job.
	// at this point the Run() can hang and caller have to implement the logic that will kill
	// this controller (SIGKILL).
	klog.Infof("Shutting down %s ...", c.name)
}

func (c *baseController) Sync(ctx context.Context, syncCtx framework.Context) error {
	return c.sync(ctx, syncCtx)
}

// runWorker runs a single worker
// The worker is asked to terminate when the passed context is cancelled and is given terminationGraceDuration time
// to complete its shutdown.
func (c *baseController) runWorker(queueCtx context.Context) {
	wait.UntilWithContext(
		queueCtx,
		func(queueCtx context.Context) {
			defer utilruntime.HandleCrash(func(in interface{}) {
				if c.syncPanicHandler == nil {
					panic(in)
				}
				if err := c.syncPanicHandler(in); err != nil {
					klog.Warningf("PANIC: Detected panic() in controller %q failed to run panic handler: %v\n\n%s", c.Name(), err, in)
				}
			})
			for {
				select {
				case <-queueCtx.Done():
					return
				default:
					c.processNextWorkItem(queueCtx)
				}
			}
		},
		1*time.Second)
}

// reconcile wraps the sync() call and if operator client is set, it handle the degraded condition if sync() returns an error.
func (c *baseController) reconcile(ctx context.Context, syncCtx framework.Context) error {
	err := c.sync(ctx, syncCtx)
	if errors.Is(err, SyntheticRequeueError) {
		return err
	}
	if c.syncErrorHandler != nil {
		if handlerErr := c.syncErrorHandler(err); handlerErr != nil {
			panic(handlerErr)
		}
	}
	return err
}

func (c *baseController) processNextWorkItem(queueCtx context.Context) {
	key, quit := c.syncContext.Queue().Get()
	if quit {
		return
	}
	defer c.syncContext.Queue().Done(key)

	stringKey, ok := key.(string)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("%q controller failed to process key %q (not a string)", c.name, key))
	}

	if err := c.reconcile(queueCtx, c.syncContext.WithQueueKey(stringKey)); err != nil {
		if errors.Is(err, SyntheticRequeueError) {
			// logging this helps detecting wedged controllers with missing pre-requirements
			klog.V(5).Infof("%q controller requested synthetic requeue with key %q", c.name, key)
		} else {
			if klog.V(4).Enabled() || key != "key" {
				utilruntime.HandleError(fmt.Errorf("%q controller failed to sync %q, err: %w", c.name, key, err))
			} else {
				utilruntime.HandleError(fmt.Errorf("%s reconciliation failed: %w", c.name, err))
			}
		}
		c.syncContext.Queue().AddRateLimited(key)
		return
	}

	c.syncContext.Queue().Forget(key)
}
