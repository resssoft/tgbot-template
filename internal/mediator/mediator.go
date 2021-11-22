package mediator

import (
	"fmt"
	"github.com/resssoft/tgbot-template/internal/models"
	"sync"
)

const (
	workers      = 5
	jobsChanSize = 10000
)

type Dispatcher struct {
	jobs        chan models.Job
	events      map[models.EventName]models.Listener
	afterEvents map[models.EventName]models.EventName
	mutex       *sync.Mutex
}

func NewDispatcher() *Dispatcher {
	d := &Dispatcher{
		jobs:        make(chan models.Job, jobsChanSize),
		events:      make(map[models.EventName]models.Listener),
		afterEvents: make(map[models.EventName]models.EventName),
		mutex:       &sync.Mutex{},
	}
	for i := 0; i < workers; i++ {
		go d.consume()
	}
	return d
}

func (d *Dispatcher) GetEvent(name models.EventName) (models.Listener, bool) {
	d.mutex.Lock()
	result, ok := d.events[name]
	d.mutex.Unlock()
	return result, ok
}

func (d *Dispatcher) SetEvent(name models.EventName, listener models.Listener) {
	d.mutex.Lock()
	d.events[name] = listener
	d.mutex.Unlock()
}

func (d *Dispatcher) GetAfterEvent(name models.EventName) (models.EventName, bool) {
	d.mutex.Lock()
	result, ok := d.afterEvents[name]
	d.mutex.Unlock()
	return result, ok
}

func (d *Dispatcher) SetAfterEvent(name models.EventName, listener models.EventName) {
	d.mutex.Lock()
	d.afterEvents[name] = listener
	d.mutex.Unlock()
}

func (d *Dispatcher) Register(listener models.Listener, names ...models.EventName) error {
	for _, name := range names {
		if _, ok := d.GetEvent(name); ok {
			return fmt.Errorf("the '%s' event is already registered", name)
		}
		d.SetEvent(name, listener)
	}

	return nil
}

func (d *Dispatcher) Dispatch(name models.EventName, event interface{}) error {
	if _, ok := d.GetEvent(name); !ok {
		return fmt.Errorf("the '%s' event is not registered", name)
	}

	d.jobs <- models.Job{EventName: name, EventType: event}

	if _, ok := d.afterEvents[name]; ok {
		d.jobs <- models.Job{EventName: d.afterEvents[name], EventType: event}
	}

	return nil
}

func (d *Dispatcher) consume() {
	var listener models.Listener
	for job := range d.jobs {
		listener, _ = d.GetEvent(job.EventName)
		go listener.Listen(job.EventName, job.EventType)
	}
}
