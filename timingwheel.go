package go_localcache

import (
	"container/list"
	"fmt"
	"time"
)

const drainWorkers = 8

type (
	Execute func(key, value interface{})

	TimingWheel struct {
		interval  time.Duration
		ticker    *time.Ticker
		slots     []*list.List
		timers    *SafeMap
		tickedPos int
		numSlots  int
		execute   Execute

		setChan    chan timingEntry
		moveChan   chan baseEntry
		removeChan chan interface{}
		drainChan  chan func(key, value interface{})
		stopChan   chan struct{}
	}

	baseEntry struct {
		delay time.Duration
		key   interface{}
	}

	timingEntry struct {
		baseEntry
		value   interface{}
		circle  int
		diff    int
		removed bool
	}

	positionEntry struct {
		pos  int
		item *timingEntry
	}

	timingTask struct {
		key, value interface{}
	}
)

func NewTimingWheel(interval time.Duration, numSlots int, execute Execute) (*TimingWheel, error) {
	if interval < 0 || numSlots < 0 || execute == nil {
		return nil, fmt.Errorf("invalid param, interval: %v, numSlots: %d, execute: %p", interval, numSlots, execute)
	}

	return newTimingWheel(interval, numSlots, execute, time.NewTicker(interval))
}

func newTimingWheel(interval time.Duration, numSlots int, execute Execute, ticker *time.Ticker) (*TimingWheel, error) {
	tw := &TimingWheel{
		interval:  interval,
		ticker:    ticker,
		slots:     make([]*list.List, numSlots),
		timers:    NewSafeMap(),
		tickedPos: numSlots - 1,
		numSlots:  numSlots,
		execute:   execute,

		setChan:    make(chan timingEntry),
		moveChan:   make(chan baseEntry),
		removeChan: make(chan interface{}),
		drainChan:  make(chan func(key, value interface{})),
		stopChan:   make(chan struct{}),
	}

	tw.initSlots()
	go tw.run()

	return tw, nil
}

func (tw *TimingWheel) initSlots() {
	for i, _ := range tw.slots {
		tw.slots[i] = list.New()
	}
}

func (tw *TimingWheel) run() {
	for {
		select {
		case <-tw.ticker.C:

		}
	}
}

func (tw *TimingWheel) onTick() {
	tw.tickedPos = (tw.tickedPos + 1) % tw.numSlots
	taskList := tw.slots[tw.tickedPos]

	tw.scanAndRun(taskList)
}

func (tw *TimingWheel) scanAndRun(taskList *list.List) {
	tasks := tw.scanTask(taskList)
	tw.runTask(tasks)
}

func (tw *TimingWheel) scanTask(taskList *list.List) []timingTask {
	var tasks []timingTask

	for e := taskList.Front(); e != nil; {
		task := e.Value.(*timingEntry)
		if task.removed {
			next := e.Next()
			taskList.Remove(e)
			e = next
			continue
		} else if task.circle > 0 {
			task.circle--
			e = e.Next()
			continue
		} else if task.diff > 0 {
			next := e.Next()
			taskList.Remove(e)

			pos := (tw.tickedPos + task.diff) % tw.numSlots
			tw.slots[pos].PushBack(task)
			tw.setTimerPosition(pos, task)
			task.diff = 0
			e = next
			continue
		}

		tasks = append(tasks, timingTask{
			key:   task.key,
			value: task.value,
		})
		next := e.Next()
		taskList.Remove(e)
		tw.timers.Del(task.key)
		e = next
	}

	return tasks
}

func (tw *TimingWheel) runTask([]timingTask) {}

func (tw *TimingWheel) setTimerPosition(pos int, task *timingEntry) {
	if v, ok := tw.timers.Get(task.key); ok {
		timer := v.(*positionEntry)
		timer.item = task
		timer.pos = pos
	} else {
		tw.timers.Set(task.key, &positionEntry{
			pos:  pos,
			item: task,
		})
	}
}
