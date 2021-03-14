package manager

import (
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/pkg/errors"
	"net"
	"sync"
)

type entry struct {
	vmid  string
	pid   *actor.PID
	ip    net.IP
	ready bool
}

func (e entry) String() string {
	return fmt.Sprintf("{VMID: %s,PID: %s}", e.vmid, e.pid)
}

type db struct {
	sync.Mutex
	machines map[string]entry
}

func initdb() *db {
	return &db{
		machines: make(map[string]entry),
	}
}

func (db *db) entry(vmid string) *entry {
	db.Lock()
	defer db.Unlock()

	entry, ok := db.machines[vmid]
	if !ok {
		return nil
	}
	return &entry
}

func (db *db) entries() (result []entry) {
	db.Lock()
	defer db.Unlock()
	for _, entry := range db.machines {
		result = append(result, entry)
	}
	return
}

func (db *db) add(vmid string, pid *actor.PID, ip net.IP) error {
	db.Lock()
	defer db.Unlock()

	if vmid == "" {
		return errors.New("vmid must not be empty")
	}
	if pid == nil {
		return errors.New("pid must not be nil")
	}

	if _, ok := db.machines[vmid]; ok {
		return errors.Errorf("vmid '%s' has already been added", vmid)
	}
	db.machines[vmid] = entry{
		vmid: vmid,
		pid:  pid,
		ip:   ip,
	}
	return nil
}

func (db *db) del(vmid string) *actor.PID {
	db.Lock()
	defer db.Unlock()

	if entry, ok := db.machines[vmid]; ok {
		delete(db.machines, vmid)
		return entry.pid
	}
	return nil
}

func (db *db) ready(vmid string, ready bool) {
	db.Lock()
	defer db.Unlock()

	entry, ok := db.machines[vmid]
	if ok {
		entry.ready = ready
		db.machines[vmid] = entry
	}
}
