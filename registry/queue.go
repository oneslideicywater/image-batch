package registry

import (
	"fmt"
	"reflect"
	"sync"
)

// Queue is thread-safe
type Queue interface {
	// Enqueue push element into the end
	Enqueue(v interface{}) error
	// Dequeue  the element from the head
	Dequeue() interface{}
	// Head checks the head
	Head() interface{}
	// Size calc the number of the element
	Size() int
	// Clear all elements
	Clear()
	// Empty return true if queue is empty
	Empty() bool
	// ElementType is the type of element,which is decided when the first element is enqueue
	ElementType() reflect.Type
}


type queue struct {
	internal []interface{}
	lock sync.Mutex
	elementType reflect.Type
}

func (q *queue) Enqueue(v interface{}) error{
	q.lock.Lock()
	defer q.lock.Unlock()
	if len(q.internal) == 0{
		q.elementType=reflect.TypeOf(v)
		q.internal=append(q.internal,v)
	}else{
		// type guard
		if reflect.TypeOf(v) == q.elementType {

			q.internal=append(q.internal,v)
		}else{
			return q.typeMismatch()
		}
	}
	return nil
}

func (q *queue) Dequeue() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.Empty() {
		return nil
	}
	ret:=q.Head()
	q.internal= q.internal[1:]
	return ret
}

func (q *queue) Head() interface{} {
	if q.Empty(){
		return nil
	}else{
		return q.internal[0]
	}
}

func (q *queue) Size() int {
	return len(q.internal)
}

func (q *queue) Empty() bool{
	return  len(q.internal) ==0
}

func (q *queue) Clear() {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.internal = q.internal[:0]
}

func (q *queue) ElementType() reflect.Type {
	return q.elementType
}


func (q *queue) typeMismatch() error{
	return fmt.Errorf("type mismatch. queue require type is %s \n",q.elementType.String())
}

func NewDefaultQueue() Queue{
	return &queue{
		internal: make([]interface{},0),
	}
}