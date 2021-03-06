package filter

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phemmer/sawmill/event"
)

type captureHandler struct {
	events []*event.Event
}

func (ch *captureHandler) Event(logEvent *event.Event) error {
	ch.events = append(ch.events, logEvent)
	return nil
}

var eventCounter uint64

func makeEvent(level event.Level) *event.Event {
	eventCounter++

	callerPC, _, _, _ := runtime.Caller(1)
	callerFunc := runtime.FuncForPC(callerPC)
	callerName := callerFunc.Name()

	message := "testing " + callerName + "()"
	data := map[string]interface{}{"test": callerName}

	return event.New(eventCounter, level, message, data, false)
}

func TestEvent(t *testing.T) {
	ch := &captureHandler{}
	filter := New(ch)

	logEvent := makeEvent(0)

	filter.Event(logEvent)

	require.NotEmpty(t, ch.events)
	assert.Equal(t, logEvent, ch.events[0])
}

func TestFilter_reject(t *testing.T) {
	ch := &captureHandler{}
	filter := New(ch)

	filter.Filter(func(e *event.Event) bool { return false })

	logEvent := makeEvent(0)

	filter.Event(logEvent)

	require.Empty(t, ch.events)
}

func TestFilter_allow(t *testing.T) {
	ch := &captureHandler{}
	filter := New(ch)

	filter.Filter(func(e *event.Event) bool { return true })

	logEvent := makeEvent(0)

	filter.Event(logEvent)

	require.NotEmpty(t, ch.events)
	assert.Equal(t, logEvent, ch.events[0])
}

func TestFilter_allowReject(t *testing.T) {
	ch := &captureHandler{}
	filter := New(ch)

	filter.Filter(func(e *event.Event) bool { return true })
	filter.Filter(func(e *event.Event) bool { return false })

	logEvent := makeEvent(0)

	filter.Event(logEvent)

	require.Empty(t, ch.events)
}

func TestLevelMin(t *testing.T) {
	ch := &captureHandler{}
	filter := New(ch)

	filter.LevelMin(event.Notice)

	table := []struct {
		level   event.Level
		allowed bool
	}{
		{event.Debug, false},
		{event.Info, false},
		{event.Notice, true},
		{event.Warning, true},
		{event.Error, true},
		{event.Critical, true},
		{event.Alert, true},
		{event.Emergency, true},
	}
	for _, test := range table {
		ch.events = []*event.Event{}
		testEvent := makeEvent(test.level)
		filter.Event(testEvent)

		if test.allowed {
			if !assert.NotEmpty(t, ch.events, "%s was rejected.", test.level) {
				continue
			}
			assert.Equal(t, testEvent, ch.events[0])
		} else {
			assert.Empty(t, ch.events, "%s was allowed.", test.level)
		}
	}
}

func TestLevelMax(t *testing.T) {
	ch := &captureHandler{}
	filter := New(ch)

	filter.LevelMax(event.Notice)

	table := []struct {
		level   event.Level
		allowed bool
	}{
		{event.Debug, true},
		{event.Info, true},
		{event.Notice, true},
		{event.Warning, false},
		{event.Error, false},
		{event.Critical, false},
		{event.Alert, false},
		{event.Emergency, false},
	}
	for _, test := range table {
		ch.events = []*event.Event{}
		testEvent := makeEvent(test.level)
		filter.Event(testEvent)

		if test.allowed {
			if !assert.NotEmpty(t, ch.events, "%s was rejected.", test.level) {
				continue
			}
			assert.Equal(t, testEvent, ch.events[0])
		} else {
			assert.Empty(t, ch.events, "%s was allowed.", test.level)
		}
	}
}

func TestLevelMinMax(t *testing.T) {
	ch := &captureHandler{}
	filter := New(ch)

	filter.LevelMin(event.Notice)
	filter.LevelMax(event.Critical)

	table := []struct {
		level   event.Level
		allowed bool
	}{
		{event.Debug, false},
		{event.Info, false},
		{event.Notice, true},
		{event.Warning, true},
		{event.Error, true},
		{event.Critical, true},
		{event.Alert, false},
		{event.Emergency, false},
	}
	for _, test := range table {
		ch.events = []*event.Event{}
		testEvent := makeEvent(test.level)
		filter.Event(testEvent)

		if test.allowed {
			if !assert.NotEmpty(t, ch.events, "%s was rejected.", test.level) {
				continue
			}
			assert.Equal(t, testEvent, ch.events[0])
		} else {
			assert.Empty(t, ch.events, "%s was allowed.", test.level)
		}
	}
}

func TestDedup(t *testing.T) {
	ch := &captureHandler{}
	filter := New(ch)
	filter.Dedup()

	testEvent1 := makeEvent(event.Notice)
	filter.Event(testEvent1)
	filter.Event(testEvent1)
	testEvent1.Id = 123
	filter.Event(testEvent1)

	testEvent2 := makeEvent(event.Notice)
	testEvent2.Message = testEvent2.Message + " 2"
	filter.Event(testEvent2)

	// should have first message, "dups" message, and the second message
	assert.Equal(t, 3, len(ch.events))

	assert.Equal(t, testEvent1.Message, ch.events[0].Message)

	assert.Equal(t, "duplicates of last log event suppressed", ch.events[1].Message)
	assert.Equal(t, 2, ch.events[1].FlatFields["count"])
	assert.Equal(t, uint64(123), ch.events[1].Id)

	assert.Equal(t, testEvent2.Message, ch.events[2].Message)
}
