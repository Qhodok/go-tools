package go_tools

import (
	"encoding/json"
	"github.com/pkg/errors"
	"strconv"
	"sync"
	//"fmt"
)

type EVENT int

const (
	ADD EVENT = iota
	UPDATE
	DELETE
)

type Component struct {
	Key  string      `json:"key"`
	Data interface{} `json:"data"`
	Next *Component  `json:"next"`
	Prev *Component  `json:"prev"`
}

type Event struct {
	*Component
	Event EVENT
}

type AddElement func(key string, data interface{}, fromStorage bool)
type DeleteElement func(key string, fromStorage bool) (element interface{})

type Storage interface {
	//SetAddAndDeleteFunction(add AddElement, del DeleteElement)
	Add(key string, data []byte)
	Delete(key string)
	Update(key string, data []byte)
	Get(key string) interface{}
}

type List struct {
	container    map[string]*Component `json:"container"`
	head         *Component            `json:"head"`
	tail         *Component            `json:"tail"`
	locker       sync.Mutex            `json:"locker"`
	eventChannel chan Event
	storage      Storage
	LastProcess  string
}

func (this *List) CreateEventListener(buffer int) (int, chan Event) {
	if this.eventChannel == nil {
		this.eventChannel = make(chan Event, buffer)
	}
	return cap(this.eventChannel), this.eventChannel
}
func (this *List) SetEventListener(listener chan Event) bool {
	if this.eventChannel == nil {
		this.eventChannel = listener
		return true
	} else {
		return false
	}

}

func NewList() (list List) {
	list = List{}
	list.container = make(map[string]*Component)
	list.head = nil
	list.tail = nil
	return
}
func NewListWithStorage(storage Storage) (list List) {
	list = List{storage: storage}
	list.container = make(map[string]*Component)
	list.head = nil
	list.tail = nil
	return
}
func NewPointerList() (list *List) {
	list = &List{}
	list.container = make(map[string]*Component)
	list.head = nil
	list.tail = nil
	return
}
func NewPointerListWithStorage(storage Storage) (list *List) {
	list = &List{storage: storage}
	list.container = make(map[string]*Component)
	list.head = nil
	list.tail = nil
	return
}

func (this *List) AddLast(key string, data interface{}) error {
	this.locker.Lock()
	defer this.locker.Unlock()
	if _, ok := this.container[key]; ok {
		return errors.New("duplicate key")
	} else {
		temp := &Component{Data: data, Key: key}
		this.container[key] = temp
		if this.head == nil {
			this.head = temp
			this.tail = temp
		} else {
			this.tail.Next = temp
			temp.Prev = this.tail
			this.tail = temp
		}
		this.broadcastEvent(Event{
			Component: temp,
			Event:     ADD,
		})
		if this.storage != nil {
			b, _ := json.Marshal(data)
			this.storage.Add(key, b)
		}
		return nil
	}
}

func (this *List) AddLastOrUpdateFromStorage(key string, data interface{}, fromStorage bool) {
	this.locker.Lock()
	defer this.locker.Unlock()
	this.addLastOrUpdateNoLock(key, data, fromStorage)
}

func (this *List) DeleteFromStorage(target string, fromStorage bool) (element interface{}) {
	this.locker.Lock()
	defer this.locker.Unlock()
	this.LastProcess = "DeleteFromStorage " + target
	if data, ok := this.container[target]; ok {
		//only one data
		this.LastProcess = "DeleteFromStorage " + target + " OK 1"
		if data.Next == nil && data.Prev == nil {
			this.head = nil
			this.tail = nil
			//data is head
		} else if data.Next != nil && data.Prev == nil {
			this.head = this.head.Next
			this.head.Prev = nil
			//data is tail
		} else if data.Prev != nil && data.Next == nil {
			this.tail = this.tail.Prev
			this.tail.Next = nil
			// data in middle
		} else {
			data.Prev.Next = data.Next
			data.Next.Prev = data.Prev
		}
		this.LastProcess = "DeleteFromStorage " + target + " OK 2"
		delete(this.container, target)
		element = data.Data
		this.LastProcess = "DeleteFromStorage " + target + " OK 3"
		this.broadcastEvent(Event{
			Component: &Component{
				Key:  data.Key,
				Data: element,
			},
			Event: DELETE,
		})
		this.LastProcess = "DeleteFromStorage " + target + " OK 4"
		if !fromStorage && this.storage != nil {
			this.storage.Delete(data.Key)
		}
		this.LastProcess = "DeleteFromStorage " + target + " OK DONE"
		return
	} else {
		this.LastProcess = "DeleteFromStorage " + target + " else 4"
		if !fromStorage && this.storage != nil {
			this.storage.Delete(target)
		}
		this.LastProcess = "DeleteFromStorage " + target + " else DONE"
		return nil
	}
}

func (this *List) AddLastOrUpdate(key string, data interface{}) error {
	this.AddLastOrUpdateFromStorage(key, data, false)
	return nil
}

func (this *List) AddLastOnExistIgnore(key string, data interface{}) bool {
	this.locker.Lock()
	defer this.locker.Unlock()
	this.LastProcess = "AddLastOnExistIgnore " + key
	if _, ok := this.container[key]; ok {
		return false
	} else {
		this.LastProcess = "AddLastOnExistIgnore " + key + " NOT 1"
		temp := &Component{Data: data, Key: key}
		this.container[key] = temp
		if this.head == nil {
			this.head = temp
			this.tail = temp
		} else {
			this.tail.Next = temp
			temp.Prev = this.tail
			this.tail = temp
		}
		this.LastProcess = "AddLastOnExistIgnore " + key + " NOT 1.1 " + "event channel " + strconv.Itoa(len(this.eventChannel))
		this.broadcastEvent(Event{
			Component: temp,
			Event:     ADD,
		})
		this.LastProcess = "AddLastOnExistIgnore " + key + " NOT 2"
		if this.storage != nil {
			b, _ := json.Marshal(data)
			this.storage.Add(key, b)
		}
		this.LastProcess = "AddLastOnExistIgnore " + key + " NOT done"
		return true
	}
}

func (this *List) AddFirst(key string, data interface{}) error {
	this.locker.Lock()
	defer this.locker.Unlock()
	if _, ok := this.container[key]; ok {
		return errors.New("duplicate key")
	} else {
		temp := &Component{Data: data, Key: key}
		this.container[key] = temp
		if this.head == nil {
			this.head = temp
			this.tail = temp
		} else {
			this.head.Prev = temp
			temp.Next = this.head
			this.head = temp
		}

		this.broadcastEvent(Event{
			Component: temp,
			Event:     ADD,
		})

		if this.storage != nil {
			b, _ := json.Marshal(data)
			this.storage.Add(key, b)
		}
		return nil
	}
}

func (this *List) AddAfter(key string, data interface{}, target string) (bool, error) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if _, ok := this.container[key]; ok {
		return false, errors.New("duplicate key")
	} else {
		if target, ok := this.container[target]; ok {
			temp := &Component{Data: data, Key: key}
			this.container[key] = temp
			if target.Next == nil {
				temp.Prev = target
				target.Next = temp
				this.tail = temp
			} else {
				target.Next.Prev = temp
				temp.Next = target.Next
				temp.Prev = target
				target.Next = temp
			}
			this.broadcastEvent(Event{
				Component: temp,
				Event:     ADD,
			})

			if this.storage != nil {
				b, _ := json.Marshal(data)
				this.storage.Add(key, b)
			}
			return true, nil
		} else {
			return false, errors.New("target not found")
		}
	}
}

func (this *List) AddWithCondition(key string, data interface{}, compare func(data interface{}) bool) (result bool, err error) {
	if this.head == nil {
		this.AddFirst(key, data)
	} else if compare(this.head.Data) {
		this.AddFirst(key, data)
		result = true
	} else {
		item := this.head.Next
		for item != nil {
			if compare(item.Data) {
				this.AddBefore(key, data, item.Key)
				result = true
				break
			}
			item = item.Next
		}
		if !result {
			this.AddLast(key, data)
		}
	}
	return
}

func (this *List) AddBefore(key string, data interface{}, target string) (bool, error) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if _, ok := this.container[key]; ok {
		return false, errors.New("duplicate key")
	} else {
		if target, ok := this.container[target]; ok {
			temp := &Component{Data: data, Key: key}
			this.container[key] = temp
			temp.Next = target
			if target.Prev == nil {
				target.Prev = temp
				this.head = temp
			} else {
				temp.Prev = target.Prev
				target.Prev.Next = temp
				target.Prev = temp
			}
			this.broadcastEvent(Event{
				Component: temp,
				Event:     ADD,
			})

			if this.storage != nil {
				b, _ := json.Marshal(data)
				this.storage.Add(key, b)
			}
			return true, nil
		} else {
			return false, errors.New("target not found")
		}
	}
}

func (this *List) RemoveFirst() (key string, element interface{}) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if this.head != nil {
		data := this.head
		this.head = this.head.Next
		if this.head != nil {
			this.head.Prev = nil
		} else {
			//fmt.Println("head nil, before",data.Data,data.Key)
		}
		delete(this.container, data.Key)
		key = data.Key
		element = data.Data
		this.broadcastEvent(Event{
			Component: &Component{
				Key:  key,
				Data: element,
			},
			Event: DELETE,
		})
		if this.storage != nil {
			this.storage.Delete(key)
		}
		return
	} else {
		return "", nil
	}
}

func (this *List) RemoveLast() (key string, element interface{}) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if this.tail != nil {
		data := this.tail
		this.tail = this.tail.Prev
		if this.tail != nil {
			this.tail.Next = nil
		}
		delete(this.container, data.Key)
		key = data.Key
		element = data.Data
		this.broadcastEvent(Event{
			Component: &Component{
				Key:  key,
				Data: element,
			},
			Event: DELETE,
		})
		if this.storage != nil {
			this.storage.Delete(key)
		}
		return
	} else {
		return "", nil
	}
}

func (this *List) RemoveAfter(target string) (key string, element interface{}) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if target, ok := this.container[target]; ok {
		if target.Next != nil {
			data := target.Next
			target.Next = data.Next
			if data.Next != nil {
				target.Next.Prev = target
			} else {
				this.tail = target
			}
			delete(this.container, data.Key)
			key = data.Key
			element = data.Data
			this.broadcastEvent(Event{
				Component: &Component{
					Key:  key,
					Data: element,
				},
				Event: DELETE,
			})
			if this.storage != nil {
				this.storage.Delete(key)
			}
			return
		} else {
			return "", nil
		}
	} else {
		return "", nil
	}
}

func (this *List) RemoveBefore(target string) (key string, element interface{}) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if target, ok := this.container[target]; ok {
		if target.Prev != nil {
			data := target.Prev
			target.Prev = data.Prev
			if data.Prev != nil {
				target.Prev.Next = target
			} else {
				this.head = target
			}
			delete(this.container, data.Key)
			key = data.Key
			element = data.Data
			this.broadcastEvent(Event{
				Component: &Component{
					Key:  key,
					Data: element,
				},
				Event: DELETE,
			})
			if this.storage != nil {
				this.storage.Delete(key)
			}
			return
		} else {
			return "", nil
		}
	} else {
		return "", nil
	}
}

func (this *List) addLastOrUpdateNoLock(key string, data interface{}, fromStorage bool) {
	this.LastProcess = "AddLastOrUpdateFromStorage " + key
	if _, ok := this.container[key]; ok {
		temp := this.container[key]
		temp.Data = data
		this.container[key] = temp
		this.LastProcess = "AddLastOrUpdateFromStorage " + key + " OK 1"
		this.broadcastEvent(Event{
			Component: temp,
			Event:     UPDATE,
		})
		this.LastProcess = "AddLastOrUpdateFromStorage " + key + " OK 2"
		if !fromStorage && this.storage != nil {
			b, _ := json.Marshal(data)
			this.storage.Update(key, b)
		}
		this.LastProcess = "AddLastOrUpdateFromStorage " + key + " OK Done"
	} else {
		temp := &Component{Data: data, Key: key}
		this.container[key] = temp
		this.LastProcess = "AddLastOrUpdateFromStorage " + key + " ELSE 1"
		if this.head == nil {
			this.head = temp
			this.tail = temp
		} else {
			this.tail.Next = temp
			temp.Prev = this.tail
			this.tail = temp
		}
		this.LastProcess = "AddLastOrUpdateFromStorage " + key + " ELSE 2"
		this.broadcastEvent(Event{
			Component: temp,
			Event:     ADD,
		})
		this.LastProcess = "AddLastOrUpdateFromStorage " + key + " ELSE 3"
		if !fromStorage && this.storage != nil {
			b, _ := json.Marshal(data)
			this.storage.Add(key, b)
		}
		this.LastProcess = "AddLastOrUpdateFromStorage " + key + " ELSE done"
	}
}

func (this *List) broadcastEvent(event Event) {
	if this.eventChannel != nil {
		if (len(this.eventChannel) + (cap(this.eventChannel) / 10)) > cap(this.eventChannel) {
			go func() {
				this.eventChannel <- event
			}()
		} else {
			this.eventChannel <- event
		}
	}
}

func (this *List) Find(target string) (element interface{}) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if data, ok := this.container[target]; ok {
		element = data.Data
		return
	} else {
		//fmt.Println("belum ada di list ",target,this.storage == nil)
		if this.storage != nil {
			element = this.storage.Get(target)
			if element != nil {
				this.addLastOrUpdateNoLock(target, element, true)
			}
			return
		} else {
			return nil
		}
	}
}

func (this *List) Contents() map[string]interface{} {
	content := make(map[string]interface{})
	for k, v := range this.container {
		content[k] = v.Data
	}
	return content
}

func (this *List) Keys() (keys []string) {
	this.locker.Lock()
	defer this.locker.Unlock()
	keys = make([]string, 0)
	for k, _ := range this.container {
		keys = append(keys, k)
	}
	return
}

func (this *List) Head() (key string, element interface{}) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if this.head != nil {
		key = this.head.Key
		element = this.head.Data
		return
	} else {
		return "", nil
	}
}

func (this *List) Tail() (key string, element interface{}) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if this.tail != nil {
		key = this.tail.Key
		element = this.tail.Data
		return
	} else {
		return "", nil
	}
}

func (this *List) Next(target string) (key string, element interface{}) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if target, ok := this.container[target]; ok {
		if target.Next != nil {
			data := target.Next
			key = data.Key
			element = data.Data
			return
		} else {
			return "", nil
		}
	} else {
		return "", nil
	}
}

func (this *List) Prev(target string) (key string, element interface{}) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if target, ok := this.container[target]; ok {
		if target.Prev != nil {
			data := target.Next
			key = data.Key
			element = data.Data
			return
		} else {
			return "", nil
		}
	} else {
		return "", nil
	}
}

func (this *List) Remove(target string) (element interface{}) {
	return this.DeleteFromStorage(target, false)
}

func (this *List) Update(key string, data interface{}) error {
	this.locker.Lock()
	defer this.locker.Unlock()
	if val, ok := this.container[key]; ok {
		val.Data = data
		this.broadcastEvent(Event{
			Component: &Component{
				Key:  key,
				Data: val,
			},
			Event: UPDATE,
		})
		return nil
	} else {
		return errors.New("data not found")
	}
}

func (this *List) Size() int {
	return len(this.container)
}

func (this *List) Sort(compare func(left interface{}, right interface{}) bool) error {
	return errors.New("not_implemented")
}
