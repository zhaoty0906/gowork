package libkademlia

import (
	"bytes"
	"net"
	"strconv"
	"testing"
	//"time"
	"log"
)

func StringToIpPort(laddr string) (ip net.IP, port uint16, err error) {
	hostString, portString, err := net.SplitHostPort(laddr)
	if err != nil {
		return
	}
	ipStr, err := net.LookupHost(hostString)
	if err != nil {
		return
	}
	for i := 0; i < len(ipStr); i++ {
		ip = net.ParseIP(ipStr[i])
		if ip.To4() != nil {
			break
		}
	}
	portInt, err := strconv.Atoi(portString)
	port = uint16(portInt)
	return
}

///////////////////////////////////////////////////////////////
//                                                           //
//                      Project 2 testing                    //
//                                                           //
///////////////////////////////////////////////////////////////

//Test iterative store
//
func TestItStore(t *testing.T) {

	log.Println("Start Test Iterative Store")
	/*instance1 := NewKademlia("localhost:9200")
	instance2 := NewKademlia("localhost:9201")
	host2, port2, _ := StringToIpPort("localhost:9201")
	instance1.DoPing(host2, port2)
	contact2, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}*/

	//generate all the tree node
	tree_node := make([]*Kademlia, 16)
	NodeAddr := make([]string, 16)

	var currentID ID
	for i := 0; i < IDBytes; i++ {
		currentID[i] = uint8(0)
	}
	for i := 0; i < 16; i++ {
		address := "localhost:" + strconv.Itoa(9202+i)
		NodeAddr[i] = address
		currentID[IDBytes-1] = uint8(i)
		tree_node[i] = NewKademliaWithId(address, currentID)
		// host_number, port_number, _ := StringToIpPort(address)
		//instance2.DoPing(host_number, port_number)
	}

	// add initial relations
	// host, port of node2
	host_number, port_number, _ := StringToIpPort(NodeAddr[2])
	tree_node[3].DoPing(host_number, port_number)
	tree_node[4].DoPing(host_number, port_number)
	tree_node[6].DoPing(host_number, port_number)
	tree_node[7].DoPing(host_number, port_number)
	tree_node[12].DoPing(host_number, port_number)

	//3
	host_number, port_number, _ = StringToIpPort(NodeAddr[3])
	tree_node[5].DoPing(host_number, port_number)
	tree_node[11].DoPing(host_number, port_number)

	//4
	host_number, port_number, _ = StringToIpPort(NodeAddr[4])
	tree_node[7].DoPing(host_number, port_number)
	//6
	host_number, port_number, _ = StringToIpPort(NodeAddr[6])
	tree_node[14].DoPing(host_number, port_number)
	tree_node[12].DoPing(host_number, port_number)
	//14
	host_number, port_number, _ = StringToIpPort(NodeAddr[14])
	tree_node[15].DoPing(host_number, port_number)

	//Store Value

	//instance 1
	log.Println("**************************************instance1")
	//using node2 store KeyID:5
	//should store in 4, 6 and 7
	currentID[IDBytes-1] = uint8(5)
	key := currentID
	value := []byte("Apple")

	log.Println("store value: ", value)
	_, err := tree_node[2].DoIterativeStore(key, value)
	if err != nil {
		t.Error("Could not store value")
	}

	// Given the right keyID, it should return the value
	foundValue, err := tree_node[4].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	foundValue, err = tree_node[6].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	foundValue, err = tree_node[7].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	//instance 2
	log.Println("*******************************instance2")
	//using node2 store KeyID:8
	//should store in 3, 4, 7 and 12
	currentID[IDBytes-1] = uint8(8)
	key = currentID
	value = []byte("Boy")

	log.Println("store value: ", value)
	_, err = tree_node[2].DoIterativeStore(key, value)
	if err != nil {
		t.Error("Could not store value")
	}

	// Given the right keyID, it should return the value
	foundValue, err = tree_node[6].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	foundValue, err = tree_node[12].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	foundValue, err = tree_node[14].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	foundValue, err = tree_node[15].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	//instance 3
	log.Println("*******************************instance3")
	//using node2 store KeyID:12
	//should store in 3, 4 and 12
	currentID[IDBytes-1] = uint8(12)
	key = currentID
	value = []byte("Cat")

	log.Println("store value: ", value)
	_, err = tree_node[2].DoIterativeStore(key, value)
	if err != nil {
		t.Error("Could not store value")
	}

	// Given the right keyID, it should return the value
	foundValue, err = tree_node[6].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	foundValue, err = tree_node[12].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	foundValue, err = tree_node[14].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}
	foundValue, err = tree_node[15].LocalFindValue(key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	//Given the wrong keyID, it should return k nodes.
	/*wrongKey := NewRandomID()
	  foundValue, contacts, err = instance1.DoFindValue(contact2, wrongKey)
	  if contacts == nil || len(contacts) < 10 {
	  	t.Error("Searching for a wrong ID did not return contacts")
	  }*/
}

//Test Iterative FindValue

func TestItFindVal(t *testing.T) {

	log.Println("Start Test Iterative Find Value")

	//generate all the tree node
	tree_node := make([]*Kademlia, 16)
	NodeAddr := make([]string, 16)

	var currentID ID
	for i := 0; i < IDBytes; i++ {
		currentID[i] = uint8(0)
	}
	for i := 0; i < 16; i++ {
		address := "localhost:" + strconv.Itoa(9302+i)
		NodeAddr[i] = address
		currentID[IDBytes-1] = uint8(i)
		tree_node[i] = NewKademliaWithId(address, currentID)
		// host_number, port_number, _ := StringToIpPort(address)
		//instance2.DoPing(host_number, port_number)
	}

	//add initial relations
	//host, port of node2
	host_number, port_number, _ := StringToIpPort(NodeAddr[2])
	tree_node[3].DoPing(host_number, port_number)
	tree_node[4].DoPing(host_number, port_number)
	tree_node[6].DoPing(host_number, port_number)
	tree_node[7].DoPing(host_number, port_number)
	tree_node[12].DoPing(host_number, port_number)

	//3
	host_number, port_number, _ = StringToIpPort(NodeAddr[3])
	tree_node[5].DoPing(host_number, port_number)
	tree_node[11].DoPing(host_number, port_number)

	//4
	host_number, port_number, _ = StringToIpPort(NodeAddr[4])
	tree_node[7].DoPing(host_number, port_number)
	//6
	host_number, port_number, _ = StringToIpPort(NodeAddr[6])
	tree_node[14].DoPing(host_number, port_number)
	tree_node[12].DoPing(host_number, port_number)
	//14
	host_number, port_number, _ = StringToIpPort(NodeAddr[14])
	tree_node[15].DoPing(host_number, port_number)

	//Store Value

	//instance 1
	log.Println("****************************instance1")
	//using node2 store KeyID:5
	currentID[IDBytes-1] = uint8(5)
	key := currentID
	value1 := []byte("Apple")

	log.Println("store value: ", value1)
	_, err := tree_node[2].DoIterativeStore(key, value1)
	if err != nil {
		t.Error("Could not store value1")
	}

	//instance 2
	log.Println("****************************instance2")
	//using node2 store KeyID:8
	//should store in 3, 4, 7 and 12
	currentID[IDBytes-1] = uint8(8)
	key = currentID
	value2 := []byte("Boy")

	log.Println("store value: ", value2)
	_, err = tree_node[2].DoIterativeStore(key, value2)
	if err != nil {
		t.Error("Could not store value2")
	}

	//instance 3
	log.Println("****************************instance3")
	//using node2 store KeyID:12
	//should store in 3, 4 and 12
	currentID[IDBytes-1] = uint8(12)
	key = currentID
	value3 := []byte("Cat")

	log.Println("store value: ", value3)
	_, err = tree_node[2].DoIterativeStore(key, value3)
	if err != nil {
		t.Error("Could not store value3")
	}

	//Test do iterative find value
	log.Println("****************************findvalue")
	currentID[IDBytes-1] = uint8(5)
	key = currentID
	foundValue, err := tree_node[2].DoIterativeFindValue(currentID)
	if !bytes.Equal(foundValue, value1) {
		t.Error("Find result did not match key1")
	}

	currentID[IDBytes-1] = uint8(8)
	key = currentID
	foundValue, err = tree_node[2].DoIterativeFindValue(currentID)
	if !bytes.Equal(foundValue, value2) {
		t.Error("Find result did not match key2")
	}

	currentID[IDBytes-1] = uint8(12)
	key = currentID
	foundValue, err = tree_node[2].DoIterativeFindValue(currentID)
	if !bytes.Equal(foundValue, value3) {
		t.Error("Find result did not match key3")
	}

	//Given the wrong keyID, it should return k nodes.
	/*wrongKey := NewRandomID()
	foundValue, contacts, err = instance1.DoFindValue(contact2, wrongKey)
	if contacts == nil || len(contacts) < 10 {
		t.Error("Searching for a wrong ID did not return contacts")
	}*/
}

// test find node

func TestIFindNode(t *testing.T) {
	log.Println("start IFInd Value")
	instance1 := NewKademlia("localhost:7774")
	instance2 := NewKademlia("localhost:7775")
	instance3 := NewKademlia("localhost:9999")
	AimID := instance3.NodeID
	log.Println(AimID)
	host2, port2, _ := StringToIpPort("localhost:7775")
	instance1.DoPing(host2, port2)
	_, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	level_1 := make([]*Kademlia, 4)
	for i := 0; i < 4; i++ {
		address := "localhost:" + strconv.Itoa(7776+i)
		level_1[i] = NewKademlia(address)
		if i == 3 {
			level_1[i].NodeID = AimID.far()
			level_1[i].SelfContact.NodeID = AimID.far()
		}
		host_number, port_number, _ := StringToIpPort(address)
		instance2.DoPing(host_number, port_number)
	}

	level_2 := make([]*Kademlia, 3)
	for j := 0; j < 3; j++ {
		address := "localhost:" + strconv.Itoa(7776+5+j)
		level_2[j] = NewKademlia(address)
		if j == 0 {
			level_2[j].NodeID = AimID.near()
			level_2[j].SelfContact.NodeID = AimID.near()
		}
		host_number, port_number, _ := StringToIpPort(address)
		level_1[0].DoPing(host_number, port_number)
	}
	level_3 := make([]*Kademlia, 3)
	for k := 0; k < 3; k++ {
		address := "localhost:" + strconv.Itoa(7776+5+5+k)
		level_3[k] = NewKademlia(address)
		if k == 2 {
			level_3[k].NodeID = AimID
			level_3[k].SelfContact.NodeID = AimID
		}
		host_number, port_number, _ := StringToIpPort(address)
		level_2[0].DoPing(host_number, port_number)
	}
	key := AimID // one node in level 3
	contacts, err := instance1.DoIterativeFindNode(key)
	log.Println(key)
	log.Println("the returning contacts in finditerativenode test: ", len(contacts))
	for _, con := range contacts {
		log.Println(con.NodeID)
	}
	if err != nil {
		t.Error("error doing DoIterativeFindNode")
	}
	if len(contacts) == 0 {
		t.Error("no constacts returned")
	}
	for _, aim := range contacts {
		if aim.NodeID.Equals(key) {
			log.Println("key found OK")
			if len(contacts) >= 8 { // 15=level_1+level_2+level_3
				log.Println("shortlist length ok")
				return
			} else {
				t.Error("forget to send finnode to the rest in the shortlist")
			}
		}
	}
	t.Error("key not found")
	return
}

func advanced_TestIFindNode(t *testing.T) {
	log.Println("start IFInd Value")
	instance1 := NewKademlia("localhost:7774")
	instance2 := NewKademlia("localhost:7775")
	instance3 := NewKademlia("localhost:9999")
	AimID := instance3.NodeID
	log.Println(AimID)
	host2, port2, _ := StringToIpPort("localhost:7775")
	instance1.DoPing(host2, port2)
	_, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	level_1 := make([]*Kademlia, 20)
	for i := 0; i < 10; i++ {
		address := "localhost:" + strconv.Itoa(6776+i)
		level_1[i] = NewKademlia(address)

		level_1[i].NodeID = AimID.advanced_far(i)
		level_1[i].SelfContact.NodeID = AimID.advanced_far(i)

		host_number, port_number, _ := StringToIpPort(address)
		instance2.DoPing(host_number, port_number)
	}

	level_2 := make([]*Kademlia, 20)
	for j := 0; j < 3; j++ {
		address := "localhost:" + strconv.Itoa(6776+20+j)
		level_2[j] = NewKademlia(address)

		level_2[j].NodeID = AimID.advanced_far(20 + j)
		level_2[j].SelfContact.NodeID = AimID.advanced_far(20 + j)

		host_number, port_number, _ := StringToIpPort(address)
		level_1[0].DoPing(host_number, port_number)
	}
	level_3 := make([]*Kademlia, 20)
	for k := 0; k < 3; k++ {
		address := "localhost:" + strconv.Itoa(6776+20+20+k)
		level_3[k] = NewKademlia(address)
		if k == 2 {
			level_3[k].NodeID = AimID
			level_3[k].SelfContact.NodeID = AimID
		}
		host_number, port_number, _ := StringToIpPort(address)
		level_2[0].DoPing(host_number, port_number)
	}
	key := AimID // one node in level 3
	contacts, err := instance1.DoIterativeFindNode(key)
	log.Println(key)
	log.Println("the returning contacts in finditerativenode test: ", len(contacts))
	for _, con := range contacts {
		log.Println(con.NodeID)
	}
	if err != nil {
		t.Error("error doing DoIterativeFindNode")
	}
	if len(contacts) == 0 {
		t.Error("no constacts returned")
	}
	for _, aim := range contacts {
		if aim.NodeID.Equals(key) {
			log.Println("key found OK")
			if len(contacts) >= 8 { // 15=level_1+level_2+level_3
				log.Println("shortlist length ok")
				return
			} else {
				t.Error("forget to send finnode to the rest in the shortlist")
			}
		}
	}
	t.Error("key not found")
	return
}
func (id ID) far() (ret ID) {
	for i := 0; i < IDBytes; i++ {
		ret[i] = id[i] ^ 1
	}
	return
}
func (id ID) advanced_far(k int) (ret ID) {
	for i := 0; i < IDBytes; i++ {
		if i == k {
			ret[i] = id[i] ^ 1
		} else {
			ret[i] = id[i] ^ 0
		}
	}
	return
}
func (id ID) near() (ret ID) {
	for i := 0; i < IDBytes; i++ {
		if i == IDBytes-1 {
			ret[i] = id[i] ^ 1
		} else {
			ret[i] = id[i] ^ 0
		}
	}
	return
}

//1 2 3 4 5
