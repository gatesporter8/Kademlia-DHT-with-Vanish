package kademlia

import (
	"fmt"
	"net/rpc"
	"strconv"
	"sync"
)

type KBucket struct {
	Contacts []Contact
	Locker   *sync.Mutex
}

func NewKBucket() *KBucket {
	result := new(KBucket)
	result.Contacts = make([]Contact, 0, k) // k from in kademlia.go
	result.Locker = &sync.Mutex{}
	return result
}

func (kb *KBucket) Update(contact *Contact) {
	Index := -1
	FlagExist := false
	FlagFull := false

	// is full?
	if len(kb.Contacts) == cap(kb.Contacts) {
		FlagFull = true
	} else {
		FlagFull = false
	}

	// exist?
	for Idx, CurrCont := range kb.Contacts {
		if CurrCont.NodeID.AsString() == contact.NodeID.AsString() {
			FlagExist = true
			Index = Idx
			break
		}
	}

	if FlagExist { // case1: already exist
		fmt.Printf("Updating Contact: " + contact.NodeID.AsString() + "\n")
		if len(kb.Contacts) > 1 {
			kb.Move2End(Index)
		}

	} else if !FlagFull { // case2: not exist, not full
		fmt.Printf("Appending Contact: " + contact.NodeID.AsString() + "\n")
		kb.Contacts = append(kb.Contacts, *contact)
	} else { // case3: not exist but full
		fmt.Printf("Choosing between Concact: " + contact.NodeID.AsString() + ", and Concact: " + kb.Contacts[0].NodeID.AsString() + "\n")
		remoteStr := kb.Contacts[0].Host.String()
		remoteStr = remoteStr + ":" + strconv.FormatUint(uint64(kb.Contacts[0].Port), 10)
		remote, err := rpc.DialHTTP("tcp", remoteStr)
		if err != nil { // case3.1 fail to contact the first one
			kb.Contacts = append(kb.Contacts[1:], *contact)
		} else { // case3.2 successfully contacted the first one
			kb.Contacts = append(kb.Contacts[1:], kb.Contacts[0])
		}
		remote.Close()
	}
	return
}

func (kb *KBucket) Move2End(Index int) {
	contact := kb.Contacts[Index]
	tmp := append(kb.Contacts[:Index], kb.Contacts[Index+1:]...)
	tmp = append(tmp, contact)
	kb.Contacts = tmp
}
