package libkademlia

// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
	// "container/heap"
)

const (
	alpha = 3
	b     = 8 * IDBytes
	kn    = 20
)

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID      ID
	SelfContact Contact
	BucketList  []Bucket
	LocalMap    map[ID][]byte
	mrequest    chan *MapRequest
	brequest    chan *BucketRequest
	Mutex       sync.RWMutex
	VDOMap      map[ID]VanashingDataObject
}

func NewKademliaWithId(laddr string, nodeID ID) *Kademlia {
	k := new(Kademlia)
	k.NodeID = nodeID
	k.BucketList = make([]Bucket, b)
	k.LocalMap = make(map[ID][]byte)
	k.mrequest = make(chan *MapRequest)
	k.brequest = make(chan *BucketRequest)
	// TODO: Initialize other state here as you add functionality.

	// Set up RPC server
	// NOTE: KademliaRPC is just a wrapper around Kademlia. This type includes
	// the RPC functions.

	s := rpc.NewServer()
	s.Register(&KademliaRPC{k})
	hostname, port, err := net.SplitHostPort(laddr)
	if err != nil {
		return nil
	}
	s.HandleHTTP(rpc.DefaultRPCPath+port,
		rpc.DefaultDebugPath+port)
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatal("Listen: ", err)
	}

	// Run RPC server forever.
	go http.Serve(l, nil)

	// Add self contact
	hostname, port, _ = net.SplitHostPort(l.Addr().String())
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

	go k.HandleChannel()
	return k
}

func (k *Kademlia) HandleChannel() {
	for {
		select {
		case MapReq := <-k.mrequest:
			if MapReq.WR == 0 { //write to map

				log.Println("store key", MapReq.Key.AsString())
				log.Println("store value", MapReq.Value)
				log.Println("store to ID:", k.SelfContact.NodeID.AsString())

				k.LocalMap[MapReq.Key] = MapReq.Value

			} else if MapReq.WR == 1 { //read from the m
				MapReq.ReturnValue = k.LocalMap[MapReq.Key]

				if MapReq.ReturnValue == nil {
					log.Println("return0")
					MapReq.Result <- 0

				} else {
					log.Println("return1value", MapReq.ReturnValue)
					MapReq.Result <- 1
				}
			}

		case BucketReq := <-k.brequest:
			//fmt.Println("get ch")
			if BucketReq.WR == 1 { //read request, set bucket
				BucketReq.ReturnValue = k.BucketList[BucketReq.Index]

				BucketReq.Result <- 1

			} else if BucketReq.WR == 0 { //write request, update the bucket
				//log.Println("brequest work well\n")
				k.Update(BucketReq)
			}
		}
	}
}

func (k *Kademlia) Update(BucketReq *BucketRequest) {

	Distance := k.NodeID.Xor(BucketReq.contact.NodeID)

	ind := Distance.PrefixLen()

	UpdateBucket := &k.BucketList[ind]

	NewContact := BucketReq.contact

	FindRes := UpdateBucket.Find(BucketReq.contact)

	//case1: client(Contact) exist, move it to back
	if FindRes != nil && FindRes.contact.NodeID.Compare(NewContact.NodeID) == 0 { //exist
		UpdateBucket.PushBack(NewContact)
		UpdateBucket.Delete(NewContact)

	} else if UpdateBucket.size < 20 { //case2: not exist, bucket not full
		UpdateBucket.PushBack(NewContact)

		//	fmt.Println(UpdateBucket.head.contact.NodeID.AsString())

	} else if UpdateBucket.size == 20 { //case3: contact not exist bucket full, ping first node in the bucket
		node := UpdateBucket.head.contact
		con, err := k.DoPing(node.Host, node.Port)
		if err != nil {
			log.Fatal("Update Call: ", err)
		}
		if con == nil { //first node not exist, remove first and add client to last
			UpdateBucket.Delete(UpdateBucket.head.contact)
			UpdateBucket.PushBack(NewContact)

		} else { //move first to last
			firstCon := UpdateBucket.head.contact
			UpdateBucket.Delete(firstCon)
			UpdateBucket.PushBack(firstCon)
		}

	} else if UpdateBucket.size > 20 {
		os.Exit(5)
	}

}

func NewKademlia(laddr string) *Kademlia {
	return NewKademliaWithId(laddr, NewRandomID())
}

type ContactNotFoundError struct {
	id  ID
	msg string
}

func (e *ContactNotFoundError) Error() string {
	return fmt.Sprintf("%x %s", e.id, e.msg)
}

func (k *Kademlia) FindContact(nodeId ID) (*Contact, error) {
	// TODO: Search through contacts, find specified ID
	// TODO: Search through contacts, find specified ID

	if nodeId == k.SelfContact.NodeID {
		return &k.SelfContact, nil
	}
	index := nodeId.Xor(k.SelfContact.NodeID).PrefixLen()
	Breq := new(BucketRequest)
	Breq.WR = 1
	Breq.Index = index
	Breq.Result = make(chan int)
	//log.Printf("before chan")
	k.brequest <- Breq

	a := <-Breq.Result
	log.Println(a)

	head := Breq.ReturnValue.head
	//log.Println(Breq.Index)
	for i := head; i != nil; i = i.Next() {
		if i.contact.NodeID.Equals(nodeId) {
			return &i.contact, nil
		}
		//log.Printf("in loop")
	}
	return nil, &ContactNotFoundError{nodeId, "Not found"}
}

type CommandFailed struct {
	msg string
}

func (e *CommandFailed) Error() string {
	return fmt.Sprintf("%s", e.msg)
}

func (k *Kademlia) DoPing(host net.IP, port uint16) (*Contact, error) {
	// TODO: Implement
	strPort := strconv.Itoa(int(port))
	client, err := rpc.DialHTTPPath("tcp", host.String()+":"+strPort,
		rpc.DefaultRPCPath+strPort)
	if err != nil {
		log.Fatal("Client failed")
	}
	ping := new(PingMessage)
	ping.MsgID = NewRandomID()
	ping.Sender = k.SelfContact
	var pong PongMessage
	err = client.Call("KademliaRPC.Ping", ping, &pong)

	if pong.Sender.NodeID.Compare(k.SelfContact.NodeID) != 0 {

		buckRq := new(BucketRequest)
		buckRq.WR = 0
		buckRq.contact = pong.Sender
		buckRq.Result = make(chan int)
		k.brequest <- buckRq
	}

	if err != nil {
		log.Fatal("Call: ", err)
		return nil, &CommandFailed{
			"Unable to ping " + fmt.Sprintf("%s:%v", host.String(), port)}
	}
	// log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
	// log.Printf("pong msgID: %s\n\n", pong.MsgID.AsString())

	// log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
	// log.Printf("pong msgID: %s\n\n", pong.MsgID.AsString())
	return &pong.Sender, err
	// return nil, &CommandFailed{
	// 	"Unable to ping " + fmt.Sprintf("%s:%v", host.String(), port)}

}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) error {
	// TODO: Implement
	strPort := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", contact.Host.String()+":"+strPort, rpc.DefaultRPCPath+strPort)
	if err != nil {
		log.Fatal("Client failed")
	}
	request := new(StoreRequest)
	request.Sender = k.SelfContact
	request.MsgID = NewRandomID()
	request.Key = key
	request.Value = value
	var result StoreResult

	err = client.Call("KademliaRPC.Store", request, &result)

	buckRq := new(BucketRequest)
	buckRq.WR = 0
	buckRq.contact = *contact
	buckRq.Result = make(chan int)
	k.brequest <- buckRq

	if err != nil {
		log.Fatal("Call: ", err)
		return &CommandFailed{
			"Unable to call rpc store " + fmt.Sprintf("%s:%v", contact.Host.String(), contact.Port)}
	} else if result.Err != nil {
		return &CommandFailed{"err in the response of store is:" + fmt.Sprintf(" %s", result.Err)}

	} else {
		return nil
	}
	// return &CommandFailed{"Not implemented"}
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) ([]Contact, error) {
	// TODO: Implement

	strPort := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", contact.Host.String()+":"+strPort, rpc.DefaultRPCPath+strPort)
	if err != nil {
		log.Fatal("Client failed")
	}
	request := new(FindNodeRequest)
	request.MsgID = NewRandomID()
	request.Sender = k.SelfContact
	var result FindNodeResult
	err = client.Call("KademliaRPC.FindNode", request, &result)

	if err != nil {
		log.Fatal("Call: ", err)
		return nil, &CommandFailed{
			"Unable to call rpc findnode " + fmt.Sprintf("%s:%v", contact.Host.String(), contact.Port)}
	}

	buckRq := new(BucketRequest)
	buckRq.WR = 0
	buckRq.contact = *contact
	buckRq.Result = make(chan int)
	k.brequest <- buckRq

	return result.Nodes, nil

}

func (k *Kademlia) DoFindValue(contact *Contact,
	searchKey ID) (value []byte, contacts []Contact, err error) {
	// TODO: Implement
	strPort := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", contact.Host.String()+":"+strPort, rpc.DefaultRPCPath+strPort)
	if err != nil {
		log.Fatal("Client failed")
	}
	request := new(FindValueRequest)
	request.MsgID = NewRandomID()
	request.Key = searchKey
	request.Sender = k.SelfContact
	var result FindValueResult
	err = client.Call("KademliaRPC.FindValue", request, &result)

	buckRq := new(BucketRequest)
	buckRq.WR = 0
	buckRq.contact = *contact
	buckRq.Result = make(chan int)
	k.brequest <- buckRq

	if err != nil {
		log.Fatal("Call: ", err)
		return nil, nil, &CommandFailed{
			"Unable to call rpc findnode " + fmt.Sprintf("%s:%v", contact.Host.String(), contact.Port)}
	}

	return result.Value, result.Nodes, nil
	// return nil, nil, &CommandFailed{"Not implemented"}
}

func (k *Kademlia) LocalFindValue(searchKey ID) ([]byte, error) {
	// TODO: Implement

	mreq := &MapRequest{WR: 1, Key: searchKey}
	mreq.Result = make(chan int)
	k.mrequest <- mreq
	result := <-mreq.Result
	if result == 0 {
		log.Println("return0")
		return nil, &CommandFailed{"Not Found"}
	} else if result == 1 {

		log.Println(mreq.ReturnValue)
		return mreq.ReturnValue, nil
	}
	return []byte(""), &CommandFailed{"Not implemented"}
}

// For project 2!
func (k *Kademlia) DoIterativeFindNode(id ID) ([]Contact, error) {
	TargetID := id
	log.Println(TargetID)
	bufferListChan := make(chan *BufferListRequest)
	shortListChan := make(chan *ShortListRequest)
	isClosestNodeUpdated := make(chan int)
	// shortListChan := make(chan []Contact)
	go k.HandleList(TargetID, shortListChan, bufferListChan, isClosestNodeUpdated)
	// parallel sending RPC
	for {
		// check the stopFlag, if true stop the sending process
		stopFlag, _ := GetShortList(shortListChan)
		if stopFlag {
			break
		}
		// get alphaNodes from bufferList, send the request
		alphaNodes := GetBufferList(bufferListChan, alpha)
		if len(alphaNodes) == 0 {
			break
		}
		log.Println("alphaNodes len: ", len(alphaNodes))
		for _, tmpContact := range alphaNodes {
			go k.SendRequest(tmpContact, TargetID, shortListChan, bufferListChan)
		}
		// wait for response 300 ms
		time.Sleep(300 * 1e6 * time.Nanosecond)
		// check whether the closestNode changed or not
		isClosestNodeUpdated <- 1
	}
	// return the result
	_, shortList := GetShortList(shortListChan)

	// if the shortList is not equal to 20, but the process stop,
	// we should check the rest of unprobed nodes in the shortList(shortList+bufferList),
	//  to fill the probed shortList quickly
	if len(shortList) < kn {
		checkNum := kn - len(shortList)
		restNodes := GetBufferList(bufferListChan, checkNum)
		log.Println(checkNum, " restNodes num: ", len(restNodes))
		log.Println()
		for _, con := range restNodes {
			k.SendRequest(con, TargetID, shortListChan, bufferListChan)
		}
	}
	// get the shortList again
	_, shortList = GetShortList(shortListChan)
	return shortList, nil
	// return nil, &CommandFailed{"Not implemented"}
}

func (k *Kademlia) DoIterativeStore(key ID, value []byte) ([]Contact, error) {
	log.Println("store value")
	contact, err := k.DoIterativeFindNode(key)

	log.Println("Return contact")
	for _, con := range contact {
		log.Print(con.NodeID)
	}
	log.Println()
	if err != nil {
		log.Fatal("Iterative find fail in DoItStore!")
	}

	for i := 0; i < len(contact); i++ {
		err := k.DoStore(&contact[i], key, value)
		if err != nil {
			log.Println("Sotore failed in cotact ", i)
		}
	}

	return contact, nil
}
func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {
	contacts, error := k.DoIterativeFindNode(key)
	if error != nil {
		return nil, &CommandFailed{"error occurs in DoIterativeFindNode" + error.Error()}
	}
	for _, element := range contacts {
		foundValue, _, err := k.DoFindValue(&element, key)
		if err != nil {
			return nil, &CommandFailed{"DoFindValue returns an error in iterartive"}
		}
		if len(foundValue) != 0 {
			return foundValue, nil
		}
	}
	return nil, &CommandFailed{"value not found"}
}

// For project 3!
func (k *Kademlia) Vanish(vdoID ID, data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {

	vdo = k.VanishData(data, numberKeys, threshold, timeoutSeconds)

	k.Mutex.Lock()
	k.VDOMap[vdoID] = vdo
	k.Mutex.Unlock()
	return
}

func (k *Kademlia) Unvanish(NodeID ID, searchKey ID) (data []byte) {

	contact, err := k.FindContact(NodeID)
	if err != nil {
		conList, _ := k.DoIterativeFindNode(NodeID)
		contact = &conList[0]
	}

	strPort := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", contact.Host.String()+":"+strPort, rpc.DefaultRPCPath+strPort)
	if err != nil {
		log.Fatal("Client failed")
	}

	request := new(GetVDORequest)
	request.Sender = k.SelfContact
	request.VdoID = searchKey
	request.MsgID = NewRandomID()
	var result GetVDOResult
	err = client.Call("KademliaRPC.GetVDO", request, &result)

	if err != nil {
		log.Fatal("Call: ", err)
		return nil
	}

	buckRq := new(BucketRequest)
	buckRq.WR = 0
	buckRq.contact = *contact
	buckRq.Result = make(chan int)
	k.brequest <- buckRq

	data = k.UnvanishData(result.VDO)

	return
}
