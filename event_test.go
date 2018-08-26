// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAppEvents(t *testing.T) {
	testcases := []string{
		EventOnInit,
		EventOnStart,
		EventOnPreShutdown,
		EventOnPostShutdown,
	}

	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [App Events]: %s", ts.URL)
	ts.app.eventStore.PublishSync(&Event{Name: EventOnStart})
	ts.app.UnsubscribeEvent(EventOnStart, EventCallback{})

	for _, event := range testcases {
		t.Run(event+" test", func(t *testing.T) {
			performEventTest(t, ts.app, event)
		})
	}

}

func performEventTest(t *testing.T, a *app, eventName string) {
	// declare functions
	onFunc1 := func(e *Event) {
		t.Log(eventName+" Func1:", e)
	}

	onFunc2 := func(e *Event) {
		t.Log(eventName+" Func2:", e)
	}

	onFunc3 := func(e *Event) {
		t.Log(eventName+" Func3:", e)
	}

	es := a.eventStore
	assert.False(t, es.IsEventExists(eventName))

	addTestEvent(a, eventName, onFunc1)
	assert.True(t, es.IsEventExists(eventName))
	assert.Equal(t, 1, es.SubscriberCount(eventName))

	addTestEvent(a, eventName, onFunc3, 4)
	assert.Equal(t, 2, es.SubscriberCount(eventName))

	addTestEvent(a, eventName, onFunc2, 2)
	assert.Equal(t, 3, es.SubscriberCount(eventName))

	// publish 1
	es.sortAndPublishSync(&Event{Name: eventName, Data: eventName + " event published 1"})

	es.Unsubscribe(eventName, onFunc2)
	assert.Equal(t, 2, es.SubscriberCount(eventName))

	// publish 2
	a.PublishEventSync(eventName, eventName+" event published 2")

	es.Unsubscribe(eventName, onFunc1)
	assert.Equal(t, 1, es.SubscriberCount(eventName))

	// publish 3
	a.PublishEventSync(eventName, eventName+" event published 3")

	es.Unsubscribe(eventName, onFunc3)
	assert.Equal(t, 0, es.SubscriberCount(eventName))

	// EventOnInit not exists
	es.Unsubscribe(eventName, onFunc3)
}

func addTestEvent(a *app, eventName string, fn func(e *Event), priority ...int) {
	switch eventName {
	case EventOnInit:
		a.OnInit(fn, priority...)
	case EventOnStart:
		a.OnStart(fn, priority...)
	case EventOnPreShutdown:
		a.OnPreShutdown(fn, priority...)
	case EventOnPostShutdown:
		a.OnPostShutdown(fn, priority...)
	}
}

func TestEventSubscribeAndUnsubscribeAndPublish(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Event Publisher]: %s", ts.URL)

	// declare functions
	myEventFunc1 := func(e *Event) {
		t.Log("myEventFunc1:", e)
	}

	myEventFunc2 := func(e *Event) {
		t.Log("myEventFunc2:", e)
	}

	myEventFunc3 := func(e *Event) {
		t.Log("myEventFunc3:", e)
	}

	es := ts.app.eventStore

	ecb1 := EventCallback{Callback: myEventFunc1, CallOnce: true}
	assert.Equal(t, 0, es.SubscriberCount("myEvent1"))
	ts.app.SubscribeEvent("myEvent1", ecb1)
	assert.Equal(t, 1, es.SubscriberCount("myEvent1"))

	ts.app.SubscribeEvent("myEvent1", EventCallback{Callback: myEventFunc2})
	assert.Equal(t, 2, es.SubscriberCount("myEvent1"))

	assert.Equal(t, 0, es.SubscriberCount("myEvent2"))
	ts.app.SubscribeEvent("myEvent2", EventCallback{Callback: myEventFunc3})
	assert.Equal(t, 1, es.SubscriberCount("myEvent2"))

	ts.app.PublishEvent("myEvent2", "myEvent2 is fired async")
	time.Sleep(time.Millisecond * 100) // for goroutine to finish

	ts.app.UnsubscribeEvent("myEvent1", ecb1)
	assert.Equal(t, 1, es.SubscriberCount("myEvent1"))

	ts.app.PublishEvent("myEvent1", "myEvent1 is fired async")
	time.Sleep(time.Millisecond * 100) // for goroutine to finish

	ts.app.PublishEvent("myEventNotExists", nil)

	ts.app.SubscribeEvent("myEvent2", EventCallback{Callback: myEventFunc3, CallOnce: true})
	ts.app.PublishEvent("myEvent2", "myEvent2 is fired async")
	time.Sleep(time.Millisecond * 100) // for goroutine to finish

	ts.app.PublishEventSync("myEvent2", "myEvent2 is fired sync")
}
