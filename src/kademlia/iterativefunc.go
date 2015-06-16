package kademlia

import (
	"container/heap"
	"errors"
	"fmt"
	//"net"
	//"net/http"
	"log"
	"net/rpc"
	"sync"
	"time"
)

type ShortList struct { // implemented through a heap
	Contacts    []Contact
	Id          ID
	Locker      *sync.Mutex
	LookUpTable map[string]bool // actually a hash set
}

func (sl ShortList) Len() int { return len(sl.Contacts) }
func (sl ShortList) Less(i, j int) bool {
	dist1 := sl.Id.Xor(sl.Contacts[i].NodeID)
	dist2 := sl.Id.Xor(sl.Contacts[j].NodeID)
	flag := dist1.Compare(dist2)
	return flag == -1
}
func (sl ShortList) Swap(i, j int) {
	sl.Contacts[i], sl.Contacts[j] = sl.Contacts[j], sl.Contacts[i]
}

func (sl *ShortList) Push(x interface{}) {
	_, ok := sl.LookUpTable[x.(Contact).NodeID.AsString()]
	if !ok {
		sl.LookUpTable[x.(Contact).NodeID.AsString()] = true
		sl.Contacts = append(sl.Contacts, x.(Contact))
	}
}

func (sl *ShortList) Pop() interface{} {
	old := sl.Contacts
	n := len(old)
	x := old[n-1]
	sl.Contacts = old[0 : n-1]
	delete(sl.LookUpTable, x.NodeID.AsString())

	return x
}

func (K *Kademlia) InitAlphaNodes(id ID) *ShortList {
	// create an empty heap
	h := &ShortList{}
	h.Id = id
	h.LookUpTable = make(map[string]bool)
	h.Locker = &sync.Mutex{}
	heap.Init(h)
	raw_contacts := make([]Contact, 0, 160*k)
	// push every contact in kademlia's node into it
	for i := 0; i < 160; i++ {
		raw_contacts = append(raw_contacts, K.Buckets[i].Contacts...)
		//fmt.Printf("%v, %v\n", i, len(K.Buckets[i].Contacts))
	}
	tmp := &ShortList{}
	tmp.Id = id
	tmp.LookUpTable = make(map[string]bool)
	tmp.Contacts = raw_contacts
	heap.Init(tmp)

	for j := 0; j < alpha; j++ {
		if tmp.Len() > 0 {
			tmpNode := heap.Pop(tmp)
			heap.Push(h, tmpNode.(Contact))
		} else {
			break
		}
	}

	// add closed alpha nodes to the heap
	return h
}

func (k *Kademlia) GetAlphaNodes(sl *ShortList) (NodesToPing []Contact, err error) {
	if sl.Len() == 0 {
		err = errors.New("Cannot find any node")
		return
	}
	NodesToPing = make([]Contact, 0, alpha)
	for i := 0; i < alpha; i++ {
		if sl.Len() == 0 {
			break
		}
		tmpNode := heap.Pop(sl)
		NodesToPing = append(NodesToPing, tmpNode.(Contact))
	}

	for _, c := range NodesToPing {
		heap.Push(sl, c)
	}
	return
}

func (k *Kademlia) DoFindNodeWithChan(Chan_FindNode chan []Contact, contact Contact, searchKey ID) { // a wraper outside DoFindNode
	// the original DoFindNode is returning a string.... have no choice but copy the code... :-(
	peer := HostAndPortString(contact.Host, contact.Port) //create peer string for DialHTTP
	client, err := rpc.DialHTTP("tcp", peer)              //creates client connection
	if err != nil {
		return
	}
	//create findvalue struct
	request := new(FindNodeRequest)
	var result FindNodeResult
	request.Sender = k.SelfContact
	request.MsgID = NewRandomID()
	request.NodeID = searchKey
	err = client.Call("KademliaCore.FindNode", request, &result)

	if err != nil {
		fmt.Printf("Error calling FindNode RPC")
	}
	//fmt.Printf("func egegfe " + contact.NodeID.AsString() + "\n")

	Chan_FindNode <- append(result.Nodes, contact)
}

func (k *Kademlia) Contacts2String(Contacts []Contact) string {
	output := fmt.Sprintf("successfully generated the shortlist of %v nodes\n", len(Contacts))
	for _, c := range Contacts {
		output = output + fmt.Sprintf("contact in shortlist: "+c.NodeID.AsString()+"\n")
	}

	return output
}

func (k *Kademlia) DoIterativeFindNode(id ID) string {
	// For project 2!
	// to make my life easier, I wrote an *_internal function to generate the active list
	// what I do here is to convert that list to a string
	result := k.DoIterativeFindNode_Internal(id)
	return k.Contacts2String(result)
}

func (K *Kademlia) DoIterativeFindNode_Internal(id ID) []Contact {
	// step 1: init shortlist, until we have alpha contacts
	// the toping shortlist(the init alpha node to ping, and more to come)
	ShortList_ToPing := K.InitAlphaNodes(id)
	// the active shortlist(the list to return)
	ShortList_Active := &ShortList{} // init active nodes
	ShortList_Active.Id = id
	ShortList_Active.LookUpTable = make(map[string]bool)
	ShortList_Active.Locker = &sync.Mutex{}
	heap.Init(ShortList_Active)
	// look up table, to avoid
	fmt.Printf("shorlist_toping len: %v, shortlist_active len: %v\n", ShortList_ToPing.Len(), ShortList_Active.Len())

	UpdateClosest := true                 // if the closest node is updated
	Chan_Timer := make(chan bool)         // timing the rpc
	Chan_FindNode := make(chan []Contact) // returning the contacts found

	// step 2: start the loop, set the max number of loops for safety
	MaxNumLoops := 1000
	for i := 0; (i < MaxNumLoops) && (UpdateClosest); i++ {
		if ShortList_Active.Len() >= k {
			fmt.Printf("We got enouth active nodes in the shortlist")
			break
		}
		// step 2.1: select alpha uncontacted nodes from shortlist, and start alpha routine to ping them
		NodesToPing, err := K.GetAlphaNodes(ShortList_ToPing)
		if err != nil { // less than alpha nodes
			fmt.Printf("%v nodes, Less than Alpha nodes to ping\n", len(NodesToPing))
			break
		}
		// run alpha find node rpc at the same time
		fmt.Printf("# of nodes to ping this iteration: %v\n", len(NodesToPing))
		for idx, node := range NodesToPing { // for each node to ping
			fmt.Printf("Pinging node #%v\n", idx)
			fmt.Printf("Pinging node is: " + node.NodeID.AsString() + "\n")
			//go func() {
			go K.DoFindNodeWithChan(Chan_FindNode, node, id)
			//ShortList_Active.Locker.Lock()
			//heap.Push(ShortList_Active, node) // inserting the current node to active node, after we succeeded in pinging it
			//ShortList_Active.Locker.Unlock()
			//}()

		}
		// run one timer
		go func() {
			time.Sleep(300 * time.Millisecond)
			Chan_Timer <- true
		}()

		// step 2.2: call rpc FindNode and merge returned contacts(alpha * k) with our shortlists
		TimeOut := false
		var OldClosest Contact
		if ShortList_ToPing.Len() > 0 {
			OldClosest = (heap.Pop(ShortList_ToPing)).(Contact) // keep track of the old closest one
			heap.Push(ShortList_ToPing, OldClosest)
		}
		for !TimeOut {
			select {
			case <-Chan_Timer: // timed out
				TimeOut = true
			case Nodes := <-Chan_FindNode: // received new nodes from rpc
				for idx, node := range Nodes {
					if idx+1 < len(Nodes) {
						heap.Push(ShortList_ToPing, node)
					} else {
						heap.Push(ShortList_Active, node)
					}
				}
			}
		}
		// check if the closest is updated
		if ShortList_ToPing.Len() > 0 {
			NewClosest := (heap.Pop(ShortList_ToPing)).(Contact)
			heap.Push(ShortList_ToPing, NewClosest)
			UpdateClosest = (OldClosest.NodeID.Compare(NewClosest.NodeID) != 0)
		} else {
			UpdateClosest = true
		}
		fmt.Printf("shorlist_toping len: %v, shortlist_active len: %v\n", ShortList_ToPing.Len(), ShortList_Active.Len())
	}

	// step 3: return the active contacts in that shortlist
	return ShortList_Active.Contacts
}
func (K *Kademlia) DoIterativeStore(key ID, value []byte) string {
	// For project 2!
	Contacts := K.DoIterativeFindNode_Internal(key)
	output := fmt.Sprintf("")
	for idx, contact := range Contacts {
		if idx == 0 {
			output = output + fmt.Sprintf("the node to store : "+contact.NodeID.AsString()+"\n")
			output = output + K.DoStore(&contact, key, value)
		} else {
			break
		}
	}
	return output
}

/*Structurally very similar to IterativeFindNode but uses the FIND_VALUE RPC
 *Additionally, it terminates the function if the value is found with the value
 *and the contact that found the value
 */
func (K *Kademlia) DoIterativeFindValue(key ID) string {

	val, contact := K.DoIterativeFindValue_Internal(key)

	var output string

	if val != nil {
		output = fmt.Sprintf("ID: %v  Value: %v", contact[0].NodeID, val)
	} else {
		output = "ERR"
	}

	return output
}

//structure for holding the result of FIND_VALUE RPC and Contact that it is being called with
type iter_value struct {
	value_result   FindValueResult
	contact_called []Contact
	self           Contact
}

//Structurally similar to DoIterativeFindNode_Internal
func (K *Kademlia) DoIterativeFindValue_Internal(key ID) ([]byte, []Contact) {
	var value []byte
	value = nil

	// step 1: init shortlist, until we have alpha contacts
	// the toping shortlist(the init alpha node to ping, and more to come)
	ShortList_ToPing := K.InitAlphaNodes(key)
	// the active shortlist(the list to return)
	ShortList_Active := &ShortList{} // init active nodes
	ShortList_Active.Id = key
	ShortList_Active.LookUpTable = make(map[string]bool)
	heap.Init(ShortList_Active)
	// look up table, to avoid
	fmt.Printf("shorlist_toping len: %v, shortlist_active len: %v\n", ShortList_ToPing.Len(), ShortList_Active.Len())

	UpdateClosest := true                      // if the closest node is updated
	Chan_Timer := make(chan bool)              // timing the rpc
	chan_value_result := make(chan iter_value) // returning the contacts found

	// step 2: start the loop, set the max number of loops for safety
	MaxNumLoops := 1000
	for i := 0; (i < MaxNumLoops) && (UpdateClosest); i++ {
		if ShortList_Active.Len() >= k {
			fmt.Printf("We got enouth active nodes in the shortlist")
			break
		}
		// step 2.1: select alpha uncontacted nodes from shortlist, and start alpha routine to ping them
		NodesToPing, err := K.GetAlphaNodes(ShortList_ToPing)
		if err != nil { // less than alpha nodes
			fmt.Printf("%v nodes, Less than Alpha nodes to ping\n", len(NodesToPing))
			break
		}
		// run alpha find node rpc at the same time
		fmt.Printf("# of nodes to ping this iteration: %v\n", len(NodesToPing))
		for idx, node := range NodesToPing { // for each node to ping
			fmt.Printf("Pinging node #%v\n", idx)
			fmt.Printf("Pinging node is: " + node.NodeID.AsString() + "\n")

			go K.DoFindValueWithChan(chan_value_result, node, key)
		}
		// run one timer
		go func() {
			time.Sleep(300 * time.Millisecond)
			Chan_Timer <- true
		}()

		// step 2.2: call rpc FindNode and merge returned contacts(alpha * k) with our shortlists
		TimeOut := false
		var OldClosest Contact
		if ShortList_ToPing.Len() > 0 {
			OldClosest = (heap.Pop(ShortList_ToPing)).(Contact) // keep track of the old closest one
			heap.Push(ShortList_ToPing, OldClosest)
		}
		for !TimeOut {
			select {
			case <-Chan_Timer: // timed out
				TimeOut = true
			case Result := <-chan_value_result: // received new nodes from rpc
				if Result.value_result.Value != nil {

					//saves value and node that value is stored at
					value = Result.value_result.Value
					contact := Result.contact_called
					//performs DoStore on closest node that doesn't have value
					for _, node := range ShortList_Active.Contacts {
						if node.NodeID.Compare(contact[0].NodeID) != 0 {
							K.DoStore(&node, key, value)
							break
						}
					}

					//returns value and contact that has the value
					return value, contact

				} else {

					//continues with regular FindNode
					for _, node := range Result.value_result.Nodes {
						heap.Push(ShortList_ToPing, node)
					}
					heap.Push(ShortList_Active, Result.self)

				}
			}
		}
		// check if the closest is updated
		if ShortList_ToPing.Len() > 0 {
			NewClosest := (heap.Pop(ShortList_ToPing)).(Contact)
			heap.Push(ShortList_ToPing, NewClosest)
			UpdateClosest = (OldClosest.NodeID.Compare(NewClosest.NodeID) != 0)
		} else {
			UpdateClosest = true
		}
	}

	// step 3: return the active contacts with a nil value (Could not find the value)
	return value, ShortList_Active.Contacts

}

func (K *Kademlia) DoFindValueWithChan(f_value_result chan iter_value, contact Contact, key ID) {

	peer := HostAndPortString(contact.Host, contact.Port)
	client, err := rpc.DialHTTP("tcp", peer)
	if err != nil {
		log.Fatal("ERR: ", err)
	}

	request := new(FindValueRequest)
	request.Sender = K.SelfContact
	request.MsgID = NewRandomID()
	request.Key = key

	var result FindValueResult
	err = client.Call("KademliaCore.FindValue", request, &result)
	if err != nil {
		log.Fatal("ERR: ", err)
	}

	v_called := make([]Contact, 1, 1)
	v_called[0] = contact

	f_value_result <- iter_value{result, v_called, contact}

}
