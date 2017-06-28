package bitfields

import (
	log "github.com/romana/rlog"
//	"github.com/romana/core/common/log/trace"
//	"github.com/romana/core/common/store"
	"github.com/romana/core/common"
	"encoding/json"
	"fmt"

)

const (
	schemaBitFieldsRoot = "bitfields"
	schemaBitFieldBody = "body"
	schemaRingKey = "idring"
)

func schemaMakeBfBodyKey(name string) string {
	return fmt.Sprintf("%s/%s/%s", schemaBitFieldsRoot, name, schemaBitFieldBody)
}

func schemaMakeBfRingKey(name string) string {
	return fmt.Sprintf("%s/%s/allocations/%s", schemaBitFieldsRoot, name, schemaRingKey)
}

type BFInterface interface {
	common.RomanaEntity

	Allocate(name string) (BitAllocation, error)
}

type BitField struct {
	Kind string
	store common.Store
	UUID string
	Name string
	Offset uint
	Length uint
}

func (bf BitField) GetName() string {
	return bf.Name
}

func (bf BitField) GetUUID() string {
	return bf.UUID
}

func (bf BitField) GetKind() string {
	return bf.Kind
}

func (bf BitField) Bytes() []byte {
	b, _ := json.Marshal(bf)
	return b
}

// begin TODO for Stas, methods below are temporary before topology switched
// to BitFields
func (bf BitField) GetRomanaIP() string { return "Not Implemented" }
func (bf BitField) SetRomanaIP(ip string) { return }
func (bf BitField) GetID() uint64 { return 0 }
func (bf BitField) SetID(id uint64) { return }
// end TODO

func New(store common.Store, name string, offset uint, length uint) (BitField, error) {
	bf := BitField{
		Kind: "BitField",
		Name: name,
		Offset: offset,
		Length: length,
		store: store,
	}

	itemKey := schemaMakeBfBodyKey(name)

	fmt.Println("DEBUG: storage ", bf.store)
	err := bf.store.Put(itemKey, bf, common.Datacenter{})
	if err != nil {
		return bf, err
	}

	ringKey := schemaMakeBfRingKey(name)
	ring := common.NewIDRing()
	rw := common.RingWrapper{IR: ring}
	err = bf.store.Put(ringKey, &rw, common.Datacenter{})
	if err != nil {
		return bf, err
	}

	return bf, nil
}

func InitFromDatacenter(store common.Store, dc common.Datacenter) (map[string]BitField, error) {
	var err error
	fields := make(map[string]BitField)

	fields["prefix"], err = New(store, "prefix", 0, uint(dc.PrefixBits))
	fields["host"], err = New(store, "host", uint(dc.PrefixBits), uint(dc.PortBits))
	fields["tenant"], err = New(store, "tenant", uint(dc.PrefixBits + dc.PortBits), uint(dc.TenantBits))
	fields["segment"], err = New(store, "segment", uint(dc.PrefixBits + dc.PortBits + dc.TenantBits), uint(dc.SegmentBits))
	fields["endpoint"], err = New(store, "endpoint", uint(dc.PrefixBits + dc.PortBits + dc.TenantBits + dc.SegmentBits), uint(dc.EndpointBits))

	return fields, err
}

func (bf BitField) Allocate(name string) (BAInterface, error) {
	ringKey := schemaMakeBfRingKey(bf.Name)
	value, err := getID(ringKey, bf.store)
	if err != nil {
		return nil, err
	}

	ba := BitAllocation{
		Kind: "BitAllocation",
		Name: name,
		Parent: bf.Name,
		Value: value,
	}

	return ba, nil
}

type BAInterface interface {
	common.RomanaEntity

	GetValue() uint64
	GetParent() string
}

type BitAllocation struct {
	Name string
	Parent string
	Kind string
	UUID string
	Value uint64
}

func (ba BitAllocation) GetName() string {
	return ba.Name
}

func (ba BitAllocation) GetUUID() string {
	return ba.UUID
}

func (ba BitAllocation) GetKind() string {
	return ba.Kind
}

func (ba BitAllocation) Bytes() []byte {
	b, _ := json.Marshal(ba)
	return b
}

// begin TODO for Stas, methods below are temporary before topology switched
// to BitAllocations
func (ba BitAllocation) GetRomanaIP() string { return "Not Implemented" }
func (ba BitAllocation) SetRomanaIP(ip string) { return }
func (ba BitAllocation) GetID() uint64 { return 0 }
func (ba BitAllocation) SetID(id uint64) { return }
// end TODO

func (ba BitAllocation) GetValue() uint64 {
	return ba.Value
}

func (ba BitAllocation) GetParent() string {
	return ba.Parent
}

// getID returns the next sequential ID for the specified key.
func getID(key string, kvStore common.Store) (uint64, error) {
	log.Debugf("getID: %s", key)
	var id uint64
	lockKey := fmt.Sprintf("%s_lock", key)
	lock, err := kvStore.NewLock(lockKey)
	if err != nil {
		log.Errorf("getID: Error acquiring lock for %s: %s", lockKey, err)
		return 0, err
	}
	log.Debugf("getID: Acquired lock for %s", lockKey)
	stopChan := make(chan struct{})
	log.Debugf("getID: Attempting to lock")
	ch, err := lock.Lock(stopChan)
	if err != nil {
		log.Errorf("getID: Error locking lock %s: %s", lock, err)
		return 0, err
	}
	log.Debugf("getID: Locked.")

	// This really is just a defer of Unlock()
	// but wrapped in a function to add logging.
	defer func() {
		err := lock.Unlock()
		if err != nil {
			log.Errorf("getID: Error unlocking %s", lockKey)
		} else {
			log.Debugf("getID: Unlocked lock %s", lockKey)
		}
	}()
	log.Debugf("getID: Entering select")
	select {
	default:
		log.Debugf("getID: In default of select")
		log.Debugf("getID: Fetching %s", key)
		idRingKvPair, err := kvStore.Get(key) // Stas // TODO this is not libkv get it's romana store get, it returns ringWrapper.
		log.Debugf("getID: Got %v", idRingKvPair)
		if err != nil {
			log.Debugf("getID: Error fetching %s: %s", key, err)
			return 0, err
		}
		log.Debugf("getID: Got %v", idRingKvPair)
		idRing, ok := IDRingAccessor(idRingKvPair)
		if !ok {
			return 0, common.NewError("Not an IDRing - %s", idRingKvPair.GetKind())
		}
		log.Debugf("getID: Got %s", idRing)
		id, err = idRing.GetID()
		if err != nil {
			return 0, err
		}
		/*
		idRingBytes, err := idRing.Encode()
		if err != nil {
			return 0, err
		}
		*/
		err = kvStore.Put(key, &common.RingWrapper{IR: idRing}, common.Datacenter{})
		if err != nil {
			return 0, err
		}
		return id, nil
	case msg := <-ch:
		log.Errorf("getID: Received from channel: %s", msg)
		return id, common.NewError("Lost lock: %s", msg)
	}
}

// func IDRingAccessor(entity common.RomanaEntity) (common.IDRing, bool) {
func IDRingAccessor(entity interface{}) (common.IDRing, bool) {
	ring, ok := entity.(*common.IDRing)
	return *ring, ok
}
