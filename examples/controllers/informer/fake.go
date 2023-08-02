package informer

import (
	"sync"
	"time"

	"k8s.io/client-go/tools/cache"
)

type FakeInformer struct {
	hasSyncedDelay       time.Duration
	eventHandler         cache.ResourceEventHandler
	AddEventHandlerCount int
	HasSyncedCount       int
	sync.Mutex
}

func NewFakeInformer(hasSyncedDelay time.Duration, hasSyncedCount int) *FakeInformer {
	return &FakeInformer{hasSyncedDelay: hasSyncedDelay, HasSyncedCount: hasSyncedCount}
}

func (f *FakeInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	f.Lock()
	defer func() { f.AddEventHandlerCount++; f.Unlock() }()
	f.eventHandler = handler
	return nil, nil
}

func (f *FakeInformer) HasSynced() bool {
	f.Lock()
	defer func() { f.HasSyncedCount++; f.Unlock() }()
	time.Sleep(f.hasSyncedDelay)
	return true
}
