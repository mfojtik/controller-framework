package embedded

import (
	"context"
	"github.com/mfojtik/controller-framework/pkg/events"
	"github.com/mfojtik/controller-framework/pkg/factory"
	"github.com/mfojtik/controller-framework/pkg/framework"
)

type controller struct {
	counter int
}

func (c *controller) sync(ctx context.Context, context framework.Context) error {
	c.counter++
	return nil
}

func New(recorder events.Recorder) framework.Controller {
	c := &controller{}
	return factory.New().WithSync(c.sync).ResyncSchedule("@every 1s").ToController("embedded", recorder)
}
