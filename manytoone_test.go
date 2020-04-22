package diodes

import (
	"runtime"
	"sync"
	"testing"
	"time"
	"unsafe"
)

func TestManyToOne_Set(t *testing.T) {
	d := NewManyToOne(5, newSpyAlerter())
	data := []byte{'1'}
	for i := 0; i < 5+1; i++ { // Ensure it's ok to set more than size.
		d.Set(unsafe.Pointer(&data))
	}
}

func TestManyToOne_TryNext(t *testing.T) {
	d := NewManyToOne(5, newSpyAlerter())
	for i := 0; i < 5; i++ {
		v := i
		d.Set(unsafe.Pointer(&v))
	}
	for i := 0; i < 5; i++ {
		v, ok := d.TryNext()
		if !ok {
			t.Fatal("should ok")
		}
		if *(*int)(v) != i {
			t.Fatal("mismatch", *(*int)(v), i)
		}
	}
}

func TestManyToOne_TryNextConcurrent(t *testing.T) {

	d := NewManyToOne(runtime.NumCPU(), nil)

	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d.Set(unsafe.Pointer(&i))
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < runtime.NumCPU(); i++ {
			time.Sleep(time.Millisecond)
			_, ok := d.TryNext()
			if !ok {
				t.Fatal("should ok")
			}
		}
	}()

	wg.Wait()
}

func TestManyToOne_TryNextAhead(t *testing.T) {
	d := NewManyToOne(5, newSpyAlerter())
	for i := 0; i < 5; i++ {
		v := i
		d.Set(unsafe.Pointer(&v))
	}
	for i := 0; i < 5; i++ {
		v, ok := d.TryNext()
		if !ok {
			t.Fatal("should ok")
		}
		if *(*int)(v) != i {
			t.Fatal("mismatch", *(*int)(v), i)
		}
	}

	_, ok := d.TryNext()
	if ok {
		t.Fatal("should not ok")
	}
}

func TestManyToOne_SetAhead(t *testing.T) {

	spy := newSpyAlerter()

	d := NewManyToOne(5, spy)
	for i := 0; i < 5+1; i++ {
		v := i
		d.Set(unsafe.Pointer(&v))
	}

	v, ok := d.TryNext()
	if !ok {
		t.Fatal("should ok")
	}
	if *(*int)(v) != 5 {
		t.Fatal("mismatch")
	}

	_, ok = d.TryNext()
	if ok {
		t.Fatal("should not ok")
	}

	missed := <-spy.AlertInput.Missed
	if missed != 5 {
		t.Fatal("missed mismatch", missed)
	}
}

type spyAlerter struct {
	AlertCalled chan bool
	AlertInput  struct {
		Missed chan int
	}
}

func newSpyAlerter() *spyAlerter {
	m := &spyAlerter{}
	m.AlertCalled = make(chan bool, 100)
	m.AlertInput.Missed = make(chan int, 100)
	return m
}
func (m *spyAlerter) Alert(missed int) {
	m.AlertCalled <- true
	m.AlertInput.Missed <- missed
}
