package libkademlia

import (
	"container/heap"
	"log"
	"net/rpc"
	"strconv"
	// "container/heap"
)

func (k *Kademlia) SendRequest(tmpContact Contact, TargetID ID, shortListChan chan<- *ShortListRequest, bufferListChan chan<- *BufferListRequest) {

	client := GenerateClient(tmpContact)
	fnRequest := new(FindNodeRequest)
	fnRequest.Sender = k.SelfContact
	fnRequest.MsgID = NewRandomID()
	fnRequest.NodeID = TargetID
	var fnResult FindNodeResult
	err := client.Call("KademliaRPC.FindNode", fnRequest, &fnResult)
	log.Println("Active Contact ID: ", tmpContact.NodeID)
	if err == nil {
		// update bucket
		buckRq := new(BucketRequest)
		buckRq.WR = 0
		buckRq.contact = tmpContact
		buckRq.Result = make(chan int)
		k.brequest <- buckRq
		// the node responsed
		AddShortListReq(tmpContact, shortListChan, true)
		AddBufferListReq(fnResult.Nodes, bufferListChan)
	} else {
		AddShortListReq(tmpContact, shortListChan, false)
	}
}

func AddBufferListReq(nodes []Contact, bufferListChan chan<- *BufferListRequest) {
	bufferListReq := &BufferListRequest{
		WR:      0,
		contact: nodes,
	}
	bufferListChan <- bufferListReq
}

func AddShortListReq(con Contact, shortListChan chan<- *ShortListRequest, activeFlag bool) {
	shortListReq := &ShortListRequest{
		WR:               0,
		ResponsedContact: con,
		ActiveFlag:       activeFlag,
	}
	shortListChan <- shortListReq
}

func GetShortList(shortListChan chan<- *ShortListRequest) (bool, []Contact) {
	shortListReq := &ShortListRequest{
		WR:     1,
		Result: make(chan int),
	}
	shortListChan <- shortListReq
	<-shortListReq.Result
	return shortListReq.StopFlag, shortListReq.ReturnValue
}

func GetBufferList(bufferListChan chan<- *BufferListRequest, alpha int) []Contact {
	bufferReq := &BufferListRequest{
		WR:     1,
		Result: make(chan int),
		number: alpha,
	}
	bufferListChan <- bufferReq
	<-bufferReq.Result
	var result []Contact
	if bufferReq.contact != nil {
		result = bufferReq.contact
	}
	return result
}

func (k *Kademlia) HandleList(TargetID ID, shortListChan <-chan *ShortListRequest, bufferListChan <-chan *BufferListRequest, isClosestNodeUpdated <-chan int) {
	// stop the search process
	stopFlag := false
	// build bufferList which is the unprobed shortList
	bufferList := k.BuildBufferList(TargetID)
	if len(bufferList) == 0 {
		log.Fatal("BufferList is empty at first")
	}
	// initialize probed shortList
	var shortList []Contact
	// initialize itemIndex map, recorde the pointed of each item in the PriorityQueue
	itemIndex := make(map[string]*Item)
	for _, item := range bufferList {
		con := item.value
		itemIndex[con.NodeID.AsString()] = item
	}
	// initialize a visited map to recorde whether a request has sended to the node
	visited := make(map[string]bool)
	// track the closestNode, initialize
	oglClosestNode := bufferList[0].value
	// handle chan
	for {
		// log.Println("handle loop begin")
		// log.Println("stopFlag: ", stopFlag)
		select {
		// bufferList
		case bufferReq := <-bufferListChan:
			// log.Println("BufferListRequest: ", bufferReq.WR)
			if bufferReq.WR == 0 {
				count := 0
				// // add the returned nodes to the bufferList(unprobed shortList)
				// when a node returen k nodes, add them to the bufferList
				bufferContact := bufferReq.contact
				for _, con := range bufferContact {
					// ok is true which means the process has sended a request to the node, so ignore it
					_, ok := visited[con.NodeID.AsString()]
					if con.NodeID == k.SelfContact.NodeID || ok {
						continue
					}
					item := UpdatePriorityQueue(&bufferList, con, TargetID)
					itemIndex[con.NodeID.AsString()] = item
					visited[con.NodeID.AsString()] = false
					count += 1
				}
				log.Println("add nodes to BufferList: ", count)
				log.Println("BufferList len: ", len(bufferList))
				log.Println(bufferList)
				log.Println("shortList len: ", len(shortList))
				log.Println()
			} else if bufferReq.WR == 1 {
				num := bufferReq.number
				// select num nodes
				var returnNodes []Contact
				for _, item := range bufferList {
					if len(returnNodes) == num {
						// have found num nodes
						break
					}
					con := item.value
					if visited[con.NodeID.AsString()] {
						continue
					}
					returnNodes = append(returnNodes, con)
					visited[con.NodeID.AsString()] = true
				}
				bufferReq.contact = returnNodes
				bufferReq.Result <- 1
			}
		// check if the bufferList closestNode changed
		case _ = <-isClosestNodeUpdated:
			var bufClosestNode Contact
			if len(bufferList) != 0 {
				bufClosestNode = bufferList[0].value
				if bufClosestNode.NodeID.Equals(oglClosestNode.NodeID) {
					stopFlag = true
				} else {
					oglDis := TargetID.Xor(oglClosestNode.NodeID)
					updDis := TargetID.Xor(bufClosestNode.NodeID)
					if oglDis.Compare(updDis) == 1 {
						// maintain the oglClosestNode is the closestNode of the shortList+bufferList
						oglClosestNode = bufClosestNode
					}
				}
			} else {
				// no new return nodes
				stopFlag = true
			}

		// shortList
		case shortListReq := <-shortListChan:
			// log.Println("shortListReq: ", shortListReq.WR)
			if shortListReq.WR == 0 {
				// when a node is active, add it to the probed shortList, remove it from bufferList(unprobed shortList)
				if !stopFlag || len(shortList) < kn {
					con := shortListReq.ResponsedContact
					if shortListReq.ActiveFlag {
						shortList = append(shortList, con)
					}
					item := itemIndex[con.NodeID.AsString()]
					heap.Remove(&bufferList, item.index)
				}
				if len(shortList) == kn {
					// shortList is 20 length
					stopFlag = true
				}
			} else if shortListReq.WR == 1 {
				// get the stopFlag
				shortListReq.ReturnValue = shortList
				shortListReq.StopFlag = stopFlag
				shortListReq.Result <- 1
			}
		}
	}
}

func (k *Kademlia) BuildBufferList(TargetID ID) PriorityQueue {
	// compute the distance and find the closest Bucket
	Distance := TargetID.Xor(k.NodeID)
	ind := Distance.PrefixLen()
	buckRq := &BucketRequest{
		WR:     1,
		Index:  ind,
		Result: make(chan int),
	}
	k.brequest <- buckRq
	<-buckRq.Result
	closestBucket := buckRq.ReturnValue

	bufferList := make(PriorityQueue, 0)
	heap.Init(&bufferList)
	// initialize the bufferList, pick alpha node from the closestBucket
	k.AddBufferList(closestBucket, &bufferList, TargetID)
	// if not enougth, find another bucket to fill the bufferList
	for i := 0; i < b; i++ {
		if len(bufferList) == alpha {
			break
		}
		if i == ind {
			continue
		}
		addBuckRq := &BucketRequest{
			WR:     1,
			Index:  i,
			Result: make(chan int),
		}
		k.brequest <- addBuckRq
		<-addBuckRq.Result
		tmpBucket := addBuckRq.ReturnValue
		// add the contact of bucket to bufferList
		k.AddBufferList(tmpBucket, &bufferList, TargetID)
	}
	return bufferList
}

func (k *Kademlia) AddBufferList(closestBucket Bucket, bufferList *PriorityQueue, TargetID ID) {
	for i := 0; i < closestBucket.size; i++ {
		if len(*bufferList) == alpha {
			break
		}
		con := closestBucket.FindByIndex(i)
		// ping the contact
		err := PingContact(con)
		if err == nil {
			buckRq := new(BucketRequest)
			buckRq.WR = 0
			buckRq.contact = con
			buckRq.Result = make(chan int)
			k.brequest <- buckRq
			_ = UpdatePriorityQueue(bufferList, con, TargetID)
		}
	}
}

func UpdatePriorityQueue(bufferList *PriorityQueue, con Contact, TargetID ID) *Item {
	item := &Item{
		value:    con,
		priority: con.NodeID.Xor(TargetID),
	}
	heap.Push(bufferList, item)
	return item
}

// make sure the first alpha node is reponsed
func PingContact(con Contact) error {
	client := GenerateClient(con)
	ping := new(PingMessage)
	ping.MsgID = NewRandomID()
	ping.Sender = con
	var pong PongMessage
	err := client.Call("KademliaRPC.Ping", ping, &pong)
	return err
}

func GenerateClient(con Contact) *rpc.Client {
	port := con.Port
	host := con.Host
	strPort := strconv.Itoa(int(port))
	client, err := rpc.DialHTTPPath("tcp", host.String()+":"+strPort,
		rpc.DefaultRPCPath+strPort)
	if err != nil {
		log.Fatal("Client failed")
	}
	return client
}
