package kademlia

// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"sync"
)

const (
	alpha = 3
	b     = 8 * IDBytes
	k     = 20
)

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID      ID
	SelfContact Contact
	Buckets     []KBucket
	Values      map[ID][]byte
	VDOS_Lock   *sync.Mutex
	VDOS        map[ID]VanashingDataObject
}

func NewKademlia(laddr string) *Kademlia {
	// TODO: Initialize other state here as you add functionality.
	k := new(Kademlia)
	k.NodeID = NewRandomID()

	// Set up RPC server
	// NOTE: KademliaCore is just a wrapper around Kademlia. This type includes
	// the RPC functions.
	rpc.Register(&KademliaCore{k})
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatal("Listen: ", err)
	}
	// Run RPC server forever.
	go http.Serve(l, nil)

	// Add self contact
	hostname, port, _ := net.SplitHostPort(l.Addr().String())
	port_int, _ := strconv.Atoi(port)
	ipAddrStrings, err := net.LookupHost(hostname)
	var host net.IP
	for i := 0; i < len(ipAddrStrings); i++ {
		host = net.ParseIP(ipAddrStrings[i])
		if host.To4() != nil {
			break
		}
	}
	k.SelfContact = Contact{k.NodeID, host, uint16(port_int)}
	fmt.Printf("Self Id: " + k.NodeID.AsString() + "\n")
	// init Buckets
	k.Buckets = make([]KBucket, b)
	for i, _ := range k.Buckets {
		k.Buckets[i] = *(NewKBucket())
	}
	k.Values = make(map[ID][]byte)
	k.VDOS_Lock = &sync.Mutex{}
	k.VDOS = make(map[ID]VanashingDataObject)
	return k
}

func Update(k *Kademlia, contact *Contact) { // update the kbucket with contact
	dist := k.NodeID.Xor(contact.NodeID)
	idx := GetBucketIndex(dist)
	bucket := &k.Buckets[idx]
	bucket.Locker.Lock()
	bucket.Update(contact)
	bucket.Locker.Unlock()
}

func GetBucketIndex(distance ID) int {
	//return the leftmost bit 1: 0011 0101 -> 3
	index := 0
	for i := IDBytes - 1; i >= 0; i-- {
		for j := 7; j >= 0; j-- {
			if (distance[i]>>uint8(j))&0x1 != 0 {
				index = b - (8*i + j)
				return index
			}
		}
	}
	return index
}

type NotFoundError struct {
	id  ID
	msg string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%x %s", e.id, e.msg)
}

func (k *Kademlia) FindContact(nodeId ID) (*Contact, error) {
	// Find contact with provided ID
	if nodeId == k.SelfContact.NodeID { // basic case: nodeId = self
		return &k.SelfContact, nil
	}
	distance := k.NodeID.Xor(nodeId)
	index := GetBucketIndex(distance)
	for _, contact := range k.Buckets[index].Contacts {
		if contact.NodeID == nodeId {
			fmt.Printf("Find Contact:" + contact.NodeID.AsString() + "\n")
			return &contact, nil
		}
	}
	return nil, &NotFoundError{nodeId, "Not found"}
}

//helper to get peer in string form
func HostAndPortString(host net.IP, port uint16) (peer string) {
	host_str := host.String()
	port_uint64 := uint64(port)
	port_str := strconv.FormatUint(port_uint64, 10)
	peer = host_str + ":" + port_str
	return peer
}

// This is the function to perform the RPC
func (k *Kademlia) DoPing(host net.IP, port uint16) string {
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	peer := HostAndPortString(host, port)    //create peer string for DialHTTP
	client, err := rpc.DialHTTP("tcp", peer) //creates client connection
	if err != nil {
		log.Fatal("ERR: ", err)
	}

	defer client.Close() //close client when finished with ping and pong
	//create ping
	ping := new(PingMessage)
	ping.Sender = k.SelfContact //create sender
	ping.MsgID = NewRandomID()  //create messageID
	//create pong
	var pong PongMessage //create pong that holds value from server
	err = client.Call("KademliaCore.Ping", ping, &pong)
	if err != nil {
		log.Fatal("ERR: ", err)
	} else {
		Update(k, &pong.Sender)
	}
	//output := fmt.Sprintf("OK: %v\n", pong) // by Haomin, debugging
	output := fmt.Sprintf("ok! " + pong.Sender.NodeID.AsString())
	return output
}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) string {
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	request := new(StoreRequest)                          //create store request request struc
	var result StoreResult                                //create storeresult struc to hold return value
	peer := HostAndPortString(contact.Host, contact.Port) //create peer string for DialHTTP
	client, err := rpc.DialHTTP("tcp", peer)              //creates client connection
	if err != nil {
		log.Fatal("ERR: ", err)
	}
	//do we need hash function here???
	//create request
	request.Sender = *(contact)
	request.MsgID = NewRandomID()
	request.Key = key
	request.Value = value
	//rpc
	err = client.Call("KademliaCore.Store", request, &result)
	if err != nil {
		log.Fatal("ERR: ", err)
	}
	//output := fmt.Sprintf("OK: %v\n", result)
	output := fmt.Sprintf("ok!")
	return output
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) string {
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	peer := HostAndPortString(contact.Host, contact.Port) //create peer string for DialHTTP
	client, err := rpc.DialHTTP("tcp", peer)              //creates client connection
	if err != nil {
		log.Fatal("ERR: ", err)
	}
	//again, do we need a hash function here?
	//create findvalue struct
	request := new(FindNodeRequest)
	var result FindNodeResult
	request.Sender = k.SelfContact
	request.MsgID = NewRandomID()
	request.NodeID = searchKey
	//make call
	err = client.Call("KademliaCore.FindNode", request, &result)
	if err != nil {
		log.Fatal("ERR: ", err)
	}
	//output := fmt.Sprintf("OK: %v\n", result.Nodes) //how can I return an array after "OK: "?!
	fmt.Printf("debugging at find_node\n")
	output := fmt.Sprintf("OK: " + result.Nodes[0].NodeID.AsString() + "\n") // by Haomin, dubugging
	return output
}

func (k *Kademlia) DoVanish(VdoID ID, data []byte, numberKeys byte, threshold byte) string {
	vdo := VanishData(*k, data, numberKeys, threshold)
	k.VDOS_Lock.Lock()
	k.VDOS[VdoID] = vdo
	k.VDOS_Lock.Unlock()
	return "vanish is done"
}

func (k *Kademlia) DoUnvanish(contact *Contact, VdoID ID) string {
	peer := HostAndPortString(contact.Host, contact.Port)
	client, err := rpc.DialHTTP("tcp", peer)
	if err != nil {
		log.Fatal("ERR: ", err)
	}
	//make findvaluerequest struct
	request := new(GetVDORequest)
	request.Sender = k.SelfContact
	request.MsgID = NewRandomID()
	request.VdoID = VdoID
	var result GetVDOResult
	err = client.Call("KademliaCore.GetVDO", request, &result)
	if err != nil {
		log.Fatal("ERR: ", err)
	}
	data := UnvanishData(*k, result.VDO)
	//data := UnvanishData(*k, k.VDOS[VdoID])
	output := fmt.Sprintf("OK: the length of data is %v\n", len(data))
	return output
}

func (k *Kademlia) DoFindValue(contact *Contact, searchKey ID) string {
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	peer := HostAndPortString(contact.Host, contact.Port)
	client, err := rpc.DialHTTP("tcp", peer)
	if err != nil {
		log.Fatal("ERR: ", err)
	}
	//make findvaluerequest struct
	//do we need to hash?
	request := new(FindValueRequest)
	request.Sender = k.SelfContact
	request.MsgID = NewRandomID()
	request.Key = searchKey
	//make call
	var result FindValueResult
	err = client.Call("KademliaCore.FindValue", request, &result)
	if err != nil {
		log.Fatal("ERR: ", err)
		//should this be in "Err: <message>" format?
	}
	//output := fmt.Sprintf("Ok: %v\n", result) //still confused about the exact format we should output
	fmt.Printf("debugging at find_value\n")
	output := fmt.Sprintf("OK: " + result.Nodes[0].NodeID.AsString() + "\n")
	return output
}

func (k *Kademlia) LocalFindValue(searchKey ID) string {
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	Val := k.Values[searchKey]
	var output string
	if Val == nil {
		output = fmt.Sprintf("Err: Cannot find this value!")
	} else {
		output = fmt.Sprintf("Ok: %v\n", Val) //still confused about the exact format we should output
	}
	return output
}
