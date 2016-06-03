package libkademlia

// Contains definitions mirroring the Kademlia spec. You will need to stick
// strictly to these to be compatible with the reference implementation and
// other groups' code.

import (
	"net"
)

type KademliaRPC struct {
	kademlia *Kademlia
}

// Host identification.
type Contact struct {
	NodeID ID
	Host   net.IP
	Port   uint16
}

type MapRequest struct {
	WR          int
	Key         ID
	Value       []byte
	ReturnValue []byte
	Result      chan int
}

type BucketRequest struct {
	WR          int
	Index       int
	contact     Contact
	ReturnValue Bucket
	Result      chan int
}

type BufferListRequest struct {
	WR      int
	contact []Contact
	number  int
	Result  chan int
}

type ShortListRequest struct {
	WR               int
	ResponsedContact Contact
	ActiveFlag       bool
	StopFlag         bool
	ReturnValue      []Contact
	Result           chan int
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

func (k *KademliaRPC) Ping(ping PingMessage, pong *PongMessage) error {

	// TODO: Finish implementation
	pong.MsgID = CopyID(ping.MsgID)
	// Specify the sender
	pong.Sender = k.kademlia.SelfContact
	// Update contact, etc
	if ping.Sender.NodeID.Compare(k.kademlia.SelfContact.NodeID) != 0 {
		buckRq := new(BucketRequest)
		buckRq.WR = 0
		buckRq.contact = ping.Sender
		buckRq.Result = make(chan int)
		k.kademlia.brequest <- buckRq
	}

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

func (k *KademliaRPC) Store(req StoreRequest, res *StoreResult) error {
	// TODO: Implement.
	UpdateRequest := &BucketRequest{WR: 0, contact: req.Sender}
	k.kademlia.brequest <- UpdateRequest

	newReq := &MapRequest{WR: 0, Key: req.Key, Value: req.Value}
	k.kademlia.mrequest <- newReq

	res.MsgID = CopyID(req.MsgID)
	res.Err = nil

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

func (k *KademliaRPC) FindNode(req FindNodeRequest, res *FindNodeResult) error {
	// TODO: Implement.

	TargetID := req.NodeID
	Distance := TargetID.Xor(k.kademlia.NodeID)
	ind := Distance.PrefixLen()
	buckRq := new(BucketRequest)
	buckRq.WR = 1
	buckRq.Result = make(chan int)
	buckRq.Index = ind
	k.kademlia.brequest <- buckRq
	_ = <-buckRq.Result
	closestBucket := buckRq.ReturnValue
	if closestBucket.size < 20 {
		for i := 0; i < closestBucket.size; i++ {
			res.Nodes = append(res.Nodes, closestBucket.FindByIndex(i))
		}
		for i := 0; i < 160; i++ {
			if len(res.Nodes) == 20 {
				break
			}
			if i == ind {
				continue
			}
			buckRq := new(BucketRequest)
			buckRq.WR = 1
			buckRq.Index = i
			buckRq.Result = make(chan int)
			k.kademlia.brequest <- buckRq
			_ = <-buckRq.Result
			tmpBucket := buckRq.ReturnValue
			for j := 0; j < tmpBucket.size; j++ {
				if len(res.Nodes) == 20 {
					break
				}
				res.Nodes = append(res.Nodes, tmpBucket.FindByIndex(j))
				res.MsgID = CopyID(req.MsgID)
			}
		}
	} else {
		for i := 0; i < 20; i++ {
			res.Nodes = append(res.Nodes, closestBucket.FindByIndex(i))
			res.MsgID = CopyID(req.MsgID)
		}
	}
	return nil
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

func (k *KademliaRPC) FindValue(req FindValueRequest, res *FindValueResult) error {
	// TODO: Implement.
	UpdateRequest := &BucketRequest{WR: 0, contact: req.Sender}
	k.kademlia.brequest <- UpdateRequest

	NewRequest := &MapRequest{WR: 1, Key: req.Key}

	NewRequest.Result = make(chan int)
	k.kademlia.mrequest <- NewRequest

	result := <-NewRequest.Result

	res.Value = NewRequest.ReturnValue

	if result == 0 { //vlaue not found, return list of node
		find_node_req := FindNodeRequest{Sender: req.Sender, MsgID: req.MsgID, NodeID: req.Key}
		find_node_res := &FindNodeResult{}
		k.FindNode(find_node_req, find_node_res)
		res.Nodes = find_node_res.Nodes
	}

	res.MsgID = CopyID(req.MsgID)
	res.Err = nil

	return nil
}

// For Project 3

type GetVDORequest struct {
	Sender Contact
	VdoID  ID
	MsgID  ID
}

type GetVDOResult struct {
	MsgID ID
	VDO   VanashingDataObject
}

func (k *KademliaRPC) GetVDO(req GetVDORequest, res *GetVDOResult) error {
	// TODO: Implement.
	kd := (*k).kademlia

	buckRq := new(BucketRequest)
	buckRq.WR = 0
	buckRq.contact = req.Sender
	buckRq.Result = make(chan int)
	kd.brequest <- buckRq

	kd.Mutex.RLock()
	value, ok := kd.VDOMap[req.VdoID]
	kd.Mutex.RUnlock()

	if ok {
		res.MsgID = req.MsgID
		res.VDO = value
	}

	return nil
}
