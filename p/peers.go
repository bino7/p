package p

import (
	. "lib"
	"sync"
)

type peersEntry struct {
	peerEntry     map[string]*peer
	userPeerEntry map[string][]*peer
	mutex         sync.Mutex
}

var peers=&peersEntry{
	peerEntry:make(map[string]*peer),
	userPeerEntry:make(map[string][]*peer),
}

func (ps *peersEntry)add(p *peer) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.peerEntry[p.uuid] = p
	userPeers := ps.userPeerEntry[p.username]
	if userPeers == nil {
		userPeers = make([]*peer, 0)
		ps.userPeerEntry[p.username] = userPeers
	}
	userPeers = append(userPeers, p)
}

func (ps *peersEntry)remove(peer *peer) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	delete(ps.userPeerEntry, peer.uuid)
	userPeers := ps.userPeerEntry[peer.username]
	for i, p := range userPeers {
		if p == peer {
			userPeers = append(userPeers[:i], userPeers[i + 1:]...)
			break
		}
	}
	if len(userPeers) == 0 {
		delete(ps.userPeerEntry, peer.username)
	}
}

func (ps *peersEntry)getWithId(uuid string) *peer {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	return ps.peerEntry[uuid]
}

func (ps *peersEntry)getWithUsername(username string) []*peer {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	return ps.userPeerEntry[username]
}

func (ps *peersEntry)forward(eve *Event,uuid string)bool{
	peer:=peers.getWithId(uuid)
	if peer==nil {
		return false
	}

	peer.In() <- eve
	return true
}