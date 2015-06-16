package kademlia

// Contains definitions mirroring the Kademlia spec. You will need to stick
// strictly to these to be compatible with the reference implementation and
// other groups' code.

import (
	"fmt"
	"net"
)

type KademliaCore struct {
	kademlia *Kademlia
}

// Host identification.
type Contact struct {
	NodeID ID
	Host   net.IP
	Port   uint16
}

///////////////////////////////////////////////////////////////////////////////
// PING
///////////////////////////////////////////////////////////////////////////////
type PingMessage struct {
	Sender Contact
	MsgID  ID
}

type PongMessage struct {
	MsgID  ID
	Sender Contact
}

func (kc *KademliaCore) Ping(ping PingMessage, pong *PongMessage) error {

	pong.MsgID = CopyID(ping.MsgID)
	pong.Sender = kc.kademlia.SelfContact
	Update(kc.kademlia, &ping.Sender)

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// STORE
///////////////////////////////////////////////////////////////////////////////
type StoreRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
	Value  []byte
}

type StoreResult struct {
	MsgID ID
	Err   error
}

func (kc *KademliaCore) Store(req StoreRequest, res *StoreResult) error {
	res.MsgID = CopyID(req.MsgID)
	kc.kademlia.Values[req.Key] = req.Value
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// FIND_NODE
///////////////////////////////////////////////////////////////////////////////
type FindNodeRequest struct {
	Sender Contact
	MsgID  ID
	NodeID ID
}

type FindNodeResult struct {
	MsgID ID
	Nodes []Contact
	Err   error
}

func (kc *KademliaCore) FindNode(req FindNodeRequest, res *FindNodeResult) error {
	res.MsgID = CopyID(req.MsgID)
	res.Nodes = FindKClosestContacts(kc.kademlia, req.NodeID)
	return nil
}

func FindKClosestContacts(kademlia *Kademlia, req_id ID) (contacts []Contact) {
	contacts = make([]Contact, 0)
	distance := kademlia.NodeID.Xor(req_id)
	var search_bucket []int

	if kademlia.NodeID == req_id {
		search_bucket = make([]int, 160)
		contacts = append(contacts, kademlia.SelfContact)
		for index := range search_bucket {
			search_bucket[index] = index
		}
	} else {
		search_bucket = GetBucketsToSearch(distance)
	}

	for index := range search_bucket {
		is_k_contacts := AddNodes(kademlia, index, req_id, &contacts)
		if is_k_contacts {
			return contacts
		}
	}

	return
}

func GetBucketsToSearch(dist ID) []int {
	//creates indices on buckets to search based on which bits are set in the distance
	search_indices := make([]int, 0)
	for i := IDBytes - 1; i >= 0; i-- {
		for j := 7; j >= 0; j-- {
			if (dist[i]>>uint8(j))&0x1 != 0 {
				search_indices = append(search_indices, (8*IDBytes)-(8*i+j)-1)
			}
		}
	}
	return search_indices
}

func AddNodes(kd *Kademlia, index int, req_id ID, close_contacts *[]Contact) bool {
	is_k_contacts := false
	list_contacts := kd.Buckets[index].Contacts
	if len(list_contacts) == 0 {
		return false
	}

	for _, contact := range list_contacts {
		*close_contacts = append(*close_contacts, contact)
		if len(*close_contacts) == k {
			is_k_contacts = true
			return is_k_contacts
		}
	}
	return is_k_contacts
}

///////////////////////////////////////////////////////////////////////////////
// FIND_VALUE
///////////////////////////////////////////////////////////////////////////////
type FindValueRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
}

// If Value is nil, it should be ignored, and Nodes means the same as in a
// FindNodeResult.
type FindValueResult struct {
	MsgID ID
	Value []byte
	Nodes []Contact
	Err   error
}

func (kc *KademliaCore) FindValue(req FindValueRequest, res *FindValueResult) error {
	res.MsgID = CopyID(req.MsgID)
	if val, ok := kc.kademlia.Values[req.Key]; ok {
		res.Value = val
	} else {
		res.Value = nil
		res.Nodes = FindKClosestContacts(kc.kademlia, req.Key)
	}

	return nil
}

type GetVDORequest struct {
	Sender Contact
	MsgID  ID
	VdoID  ID
}

type GetVDOResult struct {
	MsgID ID
	VDO   VanashingDataObject
}

func (kc *KademliaCore) GetVDO(req GetVDORequest, res *GetVDOResult) error {
	// fill in
	kc.kademlia.VDOS_Lock.Lock()
	res.MsgID = CopyID(req.MsgID)
	if val, ok := kc.kademlia.VDOS[req.VdoID]; ok {
		res.VDO = val
	} else {
		fmt.Printf("not found")
	}
	kc.kademlia.VDOS_Lock.Unlock()
	return nil
}
