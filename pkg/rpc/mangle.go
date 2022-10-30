package rpc

import (
	"sync"
	"time"
)

// MarkMessage - save original message, address and ID
func MarkMessage(msg Message, addr int) {
	item := mark{msg, addr, msg.GetInt("id")}

	markMu.Lock()
	markID++
	marks[markID] = item
	msg.SetInt("id", markID)
	markMu.Unlock()
}

// FindMessage - restore original message and address
func FindMessage(msg Message) (Message, int) {
	id := msg.GetInt("id")

	markMu.Lock()
	item, ok := marks[id]
	if ok {
		delete(marks, id)
		msg.SetInt("id", item.id)
	}
	markMu.Unlock()

	return item.msg, item.addr
}

type mark struct {
	msg  Message
	addr int
	id   int
}

var marks map[int]mark
var markID, stackTop int
var markMu sync.Mutex

func MarksWorker() {
	marks = map[int]mark{}

	for {
		markMu.Lock()

		for id := range marks {
			if id <= stackTop {
				delete(marks, id)
			}
		}

		stackTop = markID

		markMu.Unlock()

		time.Sleep(time.Second * 60)
	}
}
