package informer

import (
	"context"
	"fmt"
	"github.com/mfojtik/controller-framework/pkg/events"
	"github.com/mfojtik/controller-framework/pkg/factory"
	"github.com/mfojtik/controller-framework/pkg/framework"
)

type controller struct {
	counter  int
	informer framework.Informer
}

func (c *controller) sync(ctx context.Context, context framework.Context) error {
	i := c.informer.(*FakeInformer)
	fmt.Printf("controller-framework registered events count: %d\n", i.AddEventHandlerCount)
	fmt.Printf("controller-framework hasSynced() calls count: %d\n", i.HasSyncedCount)
	return nil
}

func New(fakeInformer framework.Informer, recorder events.Recorder) framework.Controller {
	c := &controller{
		informer: fakeInformer,
	}
	return factory.New().WithSync(c.sync).WithInformers(fakeInformer).ResyncSchedule("@every 1s").ToController("informer", recorder)
}
