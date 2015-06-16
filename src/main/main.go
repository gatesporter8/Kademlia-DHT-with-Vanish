// hhf223 zrh561 rgp633

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"
)

import (
	"kademlia"
)

func main() {
	// By default, Go seeds its RNG with 1. This would cause every program to
	// generate the same sequence of IDs. Use the current nano time to
	// random numbers
	rand.Seed(time.Now().UnixNano())

	// Get the bind and connect connection strings from command-line arguments.
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		log.Fatal("Must be invoked with exactly two arguments!\n")
	}
	listenStr := args[0]
	firstPeerStr := args[1]

	// Create the Kademlia instance
	fmt.Printf("kademlia starting up!\n")
	kadem := kademlia.NewKademlia(listenStr)

	// Confirm our server is up with a PING request and then exit.
	// Your code should loop forever, reading instructions from stdin and
	// printing their results to stdout. See README.txt for more details.
	//client, err := rpc.DialHTTP("tcp", firstPeerStr)
	_, err := rpc.DialHTTP("tcp", firstPeerStr)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}

	//ping := new(kademlia.PingMessage)
	//ping.MsgID = kademlia.NewRandomID()
	//var pong kademlia.PongMessage
	//err = client.Call("KademliaCore.Ping", ping, &pong)
	//if err != nil {
	//	log.Fatal("Call: ", err)
	//}
	//log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
	//log.Printf("pong msgID: %s\n", pong.MsgID.AsString())

	in := bufio.NewReader(os.Stdin)
	quit := false
	for !quit {
		line, err := in.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		resp := executeLine(kadem, line)
		if resp == "quit" {
			quit = true
		} else if resp != "" {
			fmt.Printf("%v\n", resp)
		}
	}
}

func executeLine(k *kademlia.Kademlia, line string) (response string) {
	toks := strings.Fields(line)
	switch {
	case toks[0] == "quit":
		response = "quit"
	case toks[0] == "whoami":
		if len(toks) > 1 {
			response = "usage: whoami"
			return
		}
		response = k.NodeID.AsString()

	case toks[0] == "print_contact":
		if len(toks) < 2 || len(toks) > 2 {
			response = "usage: print_contact [nodeID]"
			return
		}
		id, err := kademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Not a valid node ID (" + toks[1] + ")"
			return
		}
		c, err := k.FindContact(id)
		if err != nil {
			response = "ERR: Unknown contact node ID"
			return
		}
		response = "OK: NodeID=" + toks[1] + "\n"
		response += "      Host=" + c.Host.String() + "\n"
		response += "      Port=" + strconv.Itoa(int(c.Port))
	case toks[0] == "ping":
		// Do a ping
		//
		// Check if toks[1] is a valid NodeID, if not, try pinging host:port
		// print an error if neither is valid
		//
		// Following lines need to be expanded

		if len(toks) < 2 || len(toks) > 2 {
			response = "usag: ping [nodeID | host:port]"
			return
		}
		id, err := kademlia.IDFromString(toks[1])
		if err != nil {
			hostname, portstr, err := net.SplitHostPort(toks[1])
			if err != nil {
				response = "ERR: Not a valid Node ID or host:port address"
				return
			}
			port, err := strconv.Atoi(portstr)
			if err != nil {
				response = "ERR: Not a valid Node ID or host:port address"
				return
			}
			ipAddrStrings, err := net.LookupHost(hostname)
			if err != nil {
				response = "ERR: Could not find the provided hostname"
				return
			}
			var host net.IP
			for i := 0; i < len(ipAddrStrings); i++ {
				host = net.ParseIP(ipAddrStrings[i])
				if host.To4() != nil {
					break
				}
			}
			response = k.DoPing(host, uint16(port))
			return
		}
		c, err := k.FindContact(id)
		if err != nil {
			response = "ERR: Not a valid Node ID or host:port address"
			return
		}
		response = k.DoPing(c.Host, c.Port)

	case toks[0] == "local_find_value":
		// print a local variable
		if len(toks) < 2 || len(toks) > 2 {
			response = "usage: local_find_value [key]"
			return
		}
		key, err := kademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[1] + ")"
			return
		}
		response = k.LocalFindValue(key)

	case toks[0] == "store":
		// Store key, value pair at NodeID
		if len(toks) < 4 || len(toks) > 4 {
			response = "usage: store [nodeID] [key] [value]"
			return
		}
		nodeId, err := kademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid node ID (" + toks[1] + ")"
			return
		}
		contact, err := k.FindContact(nodeId)
		if err != nil {
			response = "ERR: Unable to find contact with node ID (" + toks[1] + ")"
			return
		}
		key, err := kademlia.IDFromString(toks[2])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[2] + ")"
			return
		}
		value := []byte(toks[3])

		response = k.DoStore(contact, key, value)

	case toks[0] == "find_node":
		// perform a find_node RPC
		if len(toks) < 3 || len(toks) > 3 {
			response = "usage: find_node [nodeID] [key]"
			return
		}

		nodeId, err := kademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid node ID (" + toks[1] + ")"
			return
		}
		contact, err := k.FindContact(nodeId)
		if err != nil {
			response = "ERR: Unable to find contact with node ID (" + toks[1] + ")"
			return
		}
		key, err := kademlia.IDFromString(toks[2])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[2] + ")"
			return
		}
		response = k.DoFindNode(contact, key)

	case toks[0] == "find_value":
		// perform a find_value RPC
		if len(toks) < 3 || len(toks) > 3 {
			response = "usage: find_value [nodeID] [key]"
			return
		}

		nodeId, err := kademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid node ID (" + toks[1] + ")"
			return
		}
		contact, err := k.FindContact(nodeId)
		if err != nil {
			response = "ERR: Unable to find contact with node ID (" + toks[1] + ")"
			return
		}
		key, err := kademlia.IDFromString(toks[2])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[2] + ")"
			return
		}
		response = k.DoFindValue(contact, key)

	case toks[0] == "iterativeFindNode":
		// perform an iterative find node
		if len(toks) < 2 || len(toks) > 2 {
			response = "usage: iterativeFindNode [nodeID]"
			return
		}
		id, err := kademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid node ID(" + toks[1] + ")"
			return
		}
		response = k.DoIterativeFindNode(id)

	case toks[0] == "iterativeStore":
		// perform an iterative store
		if len(toks) < 3 || len(toks) > 3 {
			response = "usage: iterativeStore [key] [value]"
			return
		}
		key, err := kademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[2] + ")"
			return
		}
		response = k.DoIterativeStore(key, []byte(toks[2]))

	case toks[0] == "iterativeFindValue":
		// performa an iterative find value
		if len(toks) < 2 || len(toks) > 2 {
			response = "usage: iterativeFindValue [key]"
			return
		}
		key, err := kademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[1] + ")"
			return
		}
		response = k.DoIterativeFindValue(key)

	case toks[0] == "vanish":
		//perfom a vanish function

		if len(toks) != 5 {
			response = "usage: vanish [VDO] [data] [numberKeys] [threshold]"
			return
		}

		key, err := kademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[1] + ")"
			return
		}
		data := []byte(toks[2])
		//numberKeys := []byte(toks[3])
		//threshold := []byte(toks[4])
		N, err := strconv.Atoi(toks[3])
		if err != nil {
			response = "ERR: Provided an invalid N (" + toks[3] + ")"
			return
		}
		T, err := strconv.Atoi(toks[4])
		if err != nil {
			response = "ERR: Provided an invalid T (" + toks[4] + ")"
			return
		}

		//response = k.DoVanish(key, data, numberKeys[0], threshold[0])
		response = k.DoVanish(key, data, byte(N), byte(T))

	case toks[0] == "unvanish":

		if len(toks) != 3 {
			response = "usage: unvanish [Node] [VDO]"
			return
		}

		key_node, err_node := kademlia.IDFromString(toks[1])
		key_vdo, err_vdo := kademlia.IDFromString(toks[2])

		if err_node != nil {
			response = "ERR: Provided an invalid key (" + toks[1] + ")"
			return
		}

		if err_vdo != nil {
			response = "ERR: Provided an invalid VDO key (" + toks[2] + ")"
			return
		}

		contact, err := k.FindContact(key_node)
		if err != nil {
			response = "ERR: Contact " + toks[1] + " cannot be found"
			return
		}

		response = k.DoUnvanish(contact, key_vdo)

	default:
		response = "ERR: Unknown command"
	}
	return
}
