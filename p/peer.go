package p

import (
	. "lib"
	"net/http"
	"log"
	"github.com/dgrijalva/jwt-go"
	"time"
	"fmt"
	"github.com/gorilla/websocket"
	"math"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

type peer struct {
	*EventSystem
	uuid       string
	username   string
	conn       *websocket.Conn
	lastAccess time.Time
	done       chan bool
}

var peerTimeout = 1300
var peerTimeoutDur = func() time.Duration {
	peerTimeoutDur, err := time.ParseDuration(fmt.Sprint(peerTimeout, "s"))
	if err != nil {
		log.Fatalln(err)
	}
	return peerTimeoutDur
}()

func newPeer() *peer {
	return &peer{
		EventSystem:NewEventSystem(),
		uuid:"",
		username:"",
		conn:nil,
		lastAccess:time.Now(),
		done:make(chan bool),
	}
}
func (p *peer)ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ok, token := authenticate(w, r)
	if !ok {
		return
	}
	claims := token.Claims.(jwt.MapClaims)
	p.uuid = claims["UUID"].(string)
	p.username = claims["Username"].(string)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	p.conn = conn

	p.AddHandler(p.checkUserInfo, p.userInfo,p.userPeers, p.updateUserInfo,p.icecandidate,p.offer,p.answer)
	p.SetFinalHandler(p.writeMessage)
	p.Run()

	peers.add(p)

	go func() {
		for {
			p.readMessage()
		}
	}()

	checkChan := make(chan bool, 1)
	checkChan <- true
	for {
		select {
		case event := <-p.Out():
			log.Println("unhandled event", event)
		case <-checkChan:
			time.AfterFunc(peerTimeoutDur, func() {
				if p.lastAccess.Add(peerTimeoutDur).Before(time.Now()) {
					p.In() <- &Event{Type:"timeout", Detail:{}}
				}
				checkChan <- true
			})
		case <-p.done:
			conn.Close()
			return
		}
	}
}

func (p *peer)connect(eve *Event) (event *Event, err error) {
	uuid := eve.Detail["remote"].(string)
	if uuid == p.uuid {
		p.In() <- eve.Forward("write")
	} else {
		remote := peers.getWithId(uuid)
		if remote == nil {

		}
		remote.In() <- eve
	}
	return
}

func (p *peer)updateUserInfo(eve *Event) (event *Event, err error) {
	username := eve.Detail["username"]
	if username != p.username {
		err = ErrorForbidden
		return
	}

	u, err := user(username)
	if err != nil {
		return err
	}

	u.Username = username
	u.Email = eve.Detail["email"]
	u.Avatar = eve.Detail["avatar"]

	err = updateUser(u)
	if err != nil {
		log.Println(err)
		err = ErrUpdateUserInfoFailed
		return
	}

	event = &Event{
		Type:"updated",
		Detail:{"version":u.Version},
	}

	return
}

func (p *peer)userInfo(eve *Event) (event *Event, err error) {
	username := eve.Detail["username"]
	version := eve.Detail["version"]
	u, err := user(username)
	if err != nil {
		return err
	}
	if u.Version > version {

		event = &Event{
			Type:"userInfo",
			Detail:{"username":u.Username,
				"email":u.Email,
				"avatar":u.Avatar,
				"version":u.Version,
			},
		}
	} else {
		event = &Event{Type:"unmodified"}
	}
	return
}
func (p *peer)userPeers(eve *Event) (event *Event, err error) {
	username := eve.Detail["username"]
	userPeers := peers.getWithUsername(username)
	if userPeers==nil || len(userPeers)==0{
		event = &Event{Type:"offline", Detail:{}}
	}else{
		ps := make([]*peer, len(userPeers))
		for i, p := range userPeers {
			ps[i] = p.uuid
		}
		event = &Event{Type:"userPeers", Detail:{"peers":ps}}
	}

	return
}
func (p *peer)checkUserInfo(eve *Event) (event *Event, err error) {
	username := eve.Detail["Username"]
	if p.username != username {
		err = ErrorForbidden
		return
	}

	u, err := user(username)
	if err != nil {
		return
	}

	version := int64(eve.Detail["Version"])
	usersVersion := int64(eve.Detail["UserVersion"])

	event = &Event{Type:"userInfo", Detail:{}}
	event["version"] = math.MaxInt64(version, u.Version)
	event["usersVersion"] = math.MaxInt64(usersVersion, u.UsersVersion)
	if u.Version > version {
		event["username"] = u.Username
		event["email"] = u.Email
	}

	if u.UsersVersion > usersVersion {
		follows, err := u.following()
		if err != nil {
			log.Println(err)
			err = ErrorForbidden
			return
		}
		users := make([]string, 0)
		for _, f := range follows {
			users = append(users, f)
		}
		event["users"] = users
	}

	return
}

func (p *peer)icecandidate(eve *Event) (event *Event, err error) {
	return p.onConnectEvent(eve)
}

func (p *peer)offer(eve *Event) (event *Event, err error) {
	return p.onConnectEvent(eve)
}

func (p *peer)answer(eve *Event) (event *Event, err error) {
	return p.onConnectEvent(eve)
}

func (p *peer)onConnectEvent(eve *Event)(event *Event, err error){
	src:=eve.Detail["src"]
	dst:=eve.Detail["dst"]
	if src==p.uuid {
		if !peers.forward(eve,dst) {
			err=errors.New(fmt.Sprint(eve.Type,"-","failed"))
		}
	}
	return
}

func (p *peer)timeout(eve *Event) (event *Event, err error) {
	p.writeMessage(eve)
	p.close()
	return
}

func (p *peer)write(eve *Event) (event *Event, err error) {
	p.writeMessage(eve)
	return
}

func (p *peer) readMessage() {
	event := new(Event)
	err := p.conn.ReadJSON(event)
	if err != nil {
		event = ErrorEvent(err, nil)
	}
	p.In() <- event
	p.lastAccess = time.Now()
}

func (p *peer) err(eve *Event) (event *Event, err error) {
	event = eve.Forward("write")
	return
}

func (p *peer) writeMessage(event *Event) {
	conn := p.conn
	err := conn.WriteJSON(event)
	if err != nil {
		log.Println(err)
	}
	return
}

func (p *peer) close() {
	peers.remove(p)
	p.done <- true
	p.Stop()
}