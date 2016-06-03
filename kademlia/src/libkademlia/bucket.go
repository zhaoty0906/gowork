package libkademlia

import (
	"fmt"
)

type Element struct {
	contact        Contact
	previous, next *Element
}

type Bucket struct {
	head *Element
	tail *Element
	size int
}

func (ele *Element) Contact() *Contact {
	return &ele.contact
}

func (ele *Element) Next() *Element {
	return ele.next
}

func (ele *Element) Previous() *Element {
	return ele.previous
}

func (bucket *Bucket) First() *Element {
	return bucket.head
}

func (bucket *Bucket) Last() *Element {
	return bucket.tail
}

func (bucket *Bucket) FindByIndex(ind int) Contact {
	cur := bucket.head

	for i := 0; i < ind; i++ {
		cur = cur.next
	}
	return cur.contact
}

func (bucket *Bucket) PushBack(c Contact) {
	bucket.size++
	e := &Element{contact: c}
	if bucket.head == nil {
		bucket.head = e

	} else {
		bucket.tail.next = e
		e.previous = bucket.tail
	}
	bucket.tail = e
}

func (bucket *Bucket) Find(c Contact) *Element {
	found := false
	var res *Element = nil
	for n := bucket.First(); n != nil && !found; n = n.Next() {
		if n.contact.NodeID.Compare(c.NodeID) == 0 {
			found = true
			res = n
		} else if n.contact.NodeID.Compare(c.NodeID) == 1 {
			found = true
			res = n.Previous()
		}
	}
	return res
}

func (bucket *Bucket) Insert(c Contact) *Bucket {

	pos := bucket.Find(c)
	if pos != nil {
		if pos.Next() == nil {
			bucket.PushBack(c)
		} else {
			e := &Element{contact: c}
			e.next = pos.next
			e.previous = pos
			pos.next.previous = e
			pos.next = e
			bucket.size++
		}

	} else {
		bucket.PushBack(c)
	}
	return bucket
}

func (bucket *Bucket) Delete(c Contact) *Bucket {

	pos := bucket.Find(c)
	if pos != nil {
		if pos.contact.NodeID.Compare(c.NodeID) == 0 { //if node found
			if pos.next == nil {
				bucket.RemoveLast()

			} else if bucket.head == pos {
				bucket.head = pos.Next()
				bucket.head.previous = nil
				bucket.size--

			} else {
				pos.Next().previous = pos.Previous()
				pos.previous.next = pos.Next()
				pos.next = nil
				pos.previous = nil
				bucket.size--

			}

		}

	} else {
		fmt.Printf("In delete: Contact Not Found!")
	}
	return bucket
}

func (bucket *Bucket) RemoveLast() *Bucket {

	if bucket == nil {
		return bucket
	} else if bucket.head == bucket.tail {
		bucket.head = nil
		bucket.tail = nil
		return nil
	} else {
		pre := bucket.tail.Previous()
		pre.next = nil
		bucket.tail = pre
	}
	bucket.size--
	return bucket

}
