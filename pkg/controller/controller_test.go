package controller

import (
	"context"
	"errors"
	"fmt"
	context2 "github.com/mfojtik/controller-framework/pkg/context"

	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/tools/cache"
	//"github.com/mfojtik/controller-framework/pkg/operator/v1helpers"
	//operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/mfojtik/controller-framework/pkg/events/eventstesting"
	"github.com/mfojtik/controller-framework/pkg/framework"
)

type fakeInformer struct {
	hasSyncedDelay       time.Duration
	eventHandler         cache.ResourceEventHandler
	addEventHandlerCount int
	hasSyncedCount       int
	sync.Mutex
}

func (f *fakeInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	f.Lock()
	defer func() { f.addEventHandlerCount++; f.Unlock() }()
	f.eventHandler = handler
	return nil, nil
}

func (f *fakeInformer) HasSynced() bool {
	f.Lock()
	defer func() { f.hasSyncedCount++; f.Unlock() }()
	time.Sleep(f.hasSyncedDelay)
	return true
}

func TestBaseController_ExitOneIfCachesWontSync(t *testing.T) {
	c := &baseController{
		syncContext:           context2.New("test", eventstesting.NewTestingEventRecorder(t)),
		informerSyncedTimeout: 1 * time.Second,
		informerSynced: []cache.InformerSynced{
			func() bool {
				return false
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if os.Getenv("BE_CRASHER") == "1" {
		c.Run(ctx, 1)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestBaseController_ExitOneIfCachesWontSync")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	e, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("unexpected error %#v", err)
	}

	exitCode := e.ExitCode()
	if exitCode != 1 {
		t.Fatalf("expected exit code %d, got %d", 1, exitCode)
	}
}

func TestBaseController_ReturnOnGracefulShutdownWhileWaitingForCachesToSync(t *testing.T) {
	c := &baseController{
		syncContext:           context2.New("test", eventstesting.NewTestingEventRecorder(t)),
		informerSyncedTimeout: 666 * time.Minute,
		informerSynced: []cache.InformerSynced{
			func() bool {
				return false
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // close context immediately

	c.Run(ctx, 1)
}

func TestBaseController_Reconcile(t *testing.T) {
	c := &baseController{
		name: "TestController",
	}

	c.sync = func(ctx context.Context, controllerContext framework.Context) error {
		return nil
	}
	if err := c.reconcile(context.TODO(), context2.New("TestController", eventstesting.NewTestingEventRecorder(t))); err != nil {
		t.Fatal(err)
	}
	c.sync = func(ctx context.Context, controllerContext framework.Context) error {
		return fmt.Errorf("error")
	}
	if err := c.reconcile(context.TODO(), context2.New("TestController", eventstesting.NewTestingEventRecorder(t))); err == nil {
		t.Fatal("expected error, got none")
	}
}

func TestBaseController_ReconcileErrorAndPanicHandlers(t *testing.T) {
	errsHandled := []error{}
	c := &baseController{
		name: "TestController",
		syncErrorHandler: func(err error) error {
			errsHandled = append(errsHandled, err)
			return nil
		},
	}

	syncErr := errors.New("sync error")

	c.sync = func(ctx context.Context, controllerContext framework.Context) error {
		return syncErr
	}
	if err := c.reconcile(context.TODO(), context2.New("TestController", eventstesting.NewTestingEventRecorder(t))); !errors.Is(err, syncErr) {
		t.Fatalf("expected sync() to return original error, got %v", err)
	}
	c.reconcile(context.TODO(), context2.New("TestController", eventstesting.NewTestingEventRecorder(t)))
	if len(errsHandled) != 2 {
		t.Fatalf("expected 2 errors, got %d (%#v)", len(errsHandled), errsHandled)
	}

	c.syncErrorHandler = func(err error) error {
		return err
	}
	panicCount := 0
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicCount++
			}
		}()
		c.reconcile(context.TODO(), context2.New("TestController", eventstesting.NewTestingEventRecorder(t)))
	}()
	if panicCount == 0 {
		t.Fatalf("expected to panic when sync error handler error out")
	}
}

func TestBaseController_Run(t *testing.T) {
	informer := &fakeInformer{hasSyncedDelay: 200 * time.Millisecond}
	controllerCtx, cancel := context.WithCancel(context.Background())
	syncCount := 0
	postStartHookSyncCount := 0
	postStartHookDone := false

	c := &baseController{
		name:           "test",
		informerSynced: []cache.InformerSynced{informer.HasSynced},
		sync: func(ctx context.Context, syncCtx framework.Context) error {
			defer func() { syncCount++ }()
			defer t.Logf("Sync() call with %q", syncCtx.QueueKey())
			if syncCtx.QueueKey() == "postStartHookKey" {
				postStartHookSyncCount++
				return fmt.Errorf("test error")
			}
			return nil
		},
		syncContext: context2.New("test", eventstesting.NewTestingEventRecorder(t)),
		resyncEvery: 200 * time.Millisecond,
		postStartHooks: []framework.PostStartHook{func(ctx context.Context, syncContext framework.Context) error {
			defer func() {
				postStartHookDone = true
			}()
			syncContext.Queue().Add("postStartHookKey")
			<-ctx.Done()
			t.Logf("post start hook terminated")
			return nil
		}},
	}

	time.AfterFunc(1*time.Second, func() {
		cancel()
	})
	c.Run(controllerCtx, 1)

	informer.Lock()
	if informer.hasSyncedCount == 0 {
		t.Errorf("expected HasSynced() called at least once, got 0")
	}
	informer.Unlock()
	if syncCount == 0 {
		t.Errorf("expected at least one sync call, got 0")
	}
	if postStartHookSyncCount == 0 {
		t.Errorf("expected the post start hook queue key, got none")
	}
	if !postStartHookDone {
		t.Errorf("expected the post start hook to be terminated when context is cancelled")
	}
}
