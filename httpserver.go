package main

import (
	"container/list"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type User struct {
	uid            string
	session        string
	isOnline       bool
	GetMsgCnt      int
	ch             chan string
	MM             string
	LastAccessTime time.Time
	isVisitor      bool
}

const CHAN_BUFFER_SIZE = 5

//user = &User{uid: uid, session: "session", isOnline: false, GetMsgCnt: 0, ch: nil, MM: ""}
//user = &User{uid, "session", true, 0, nil, ""}

var UserDBLock = struct {
	sync.RWMutex
	m map[string]*User
}{m: make(map[string]*User)}

var ChatRoomLock = struct {
	sync.RWMutex
	m map[string]*list.List
}{m: make(map[string]*list.List)}

func pub_getRoomFromMap(cid string) (*list.List, bool) {
	ChatRoomLock.RWMutex.RLock()
	room, ok := ChatRoomLock.m[cid]
	ChatRoomLock.RWMutex.RUnlock()
	return room, ok
}

func pub_delRoomFromMap(cid string) {
	ChatRoomLock.RWMutex.Lock()
	delete(ChatRoomLock.m, cid)
	ChatRoomLock.RWMutex.Unlock()
}

func pub_setRoomFromMap(cid string, room *list.List) {
	ChatRoomLock.RWMutex.Lock()
	ChatRoomLock.m[cid] = room
	ChatRoomLock.RWMutex.Unlock()

}
func pub_delUserFromMap(uid string) {
	UserDBLock.RWMutex.Lock()
	delete(UserDBLock.m, uid)
	UserDBLock.RWMutex.Unlock()
}

func pub_setUserFromMap(uid string, user *User) {
	UserDBLock.RWMutex.Lock()
	UserDBLock.m[uid] = user
	UserDBLock.RWMutex.Unlock()

}

func pub_getUserFromMap(uid string) (*User, bool) {
	UserDBLock.RWMutex.RLock()
	user, ok := UserDBLock.m[uid]
	UserDBLock.RWMutex.RUnlock()
	return user, ok

}

//var UserDB map[string]*User //get obj from map[string]User will copy struct only
//var ChatRoom map[string]*list.List

func Handler(w http.ResponseWriter, r *http.Request) {

	//	fmt.Fprintf(w, "<h1>welcome to go chat server %s!</h1>", r.URL.Path[1:])
	//	return

	File := r.URL.Path[1:]
	t, err := template.ParseFiles(File)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
	return

}

func getmsg(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	callback := r.FormValue("callback")

	isVisitor := (uid == "")

	if isVisitor {
		uid = pub_GetVid(r)
	}

	rv := returnValue{IRet: 0, Cmd: "getmsg", Uid: uid}

	user, ok := pub_getUserFromMap(uid)
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("GETMSG: new user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		if callback == "" {
			fmt.Fprintf(w, pub_tostring(rv))
		} else {
			fmt.Fprintf(w, callback+"(%s)", pub_tostring(rv))
		}
		return
	}

	//Trace("GETMSG: ", uid, tuser.GetMsgCnt)
	if user.GetMsgCnt > 0 {
		Info("DAME msgcnt,2 GETMSG AT THE SAME TIME")
		for {
			if user.GetMsgCnt <= 0 {
				break
			}
			Info("GETMSG: Clean up ", uid)
			rv2 := returnValue2{IRet: 0, Cmd: "getmsg"}
			user.ch <- pub_tostring(rv2)
			runtime.Gosched()
			//runtime.Gosched()
			//tuser.GetMsgCnt = tuser.GetMsgCnt - 1
		}
	}

	user.GetMsgCnt++
	defer func() {
		user.GetMsgCnt = user.GetMsgCnt - 1
	}()

	cookie := http.Cookie{Name: "session", Value: user.session, Path: "/", HttpOnly: true}
	http.SetCookie(w, &cookie)

	if isVisitor {
		cookie = http.Cookie{Name: "vid", Value: user.uid, Path: "/"}
		http.SetCookie(w, &cookie)
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(4e9)
		timeout <- true
	}()

	select {
	case msg := <-user.ch:

		rv.IRet = 1
		rv.Msg = msg
		fmt.Fprintf(w, msg) //tuser.MM

		user.MM = ""
	case <-timeout:
		rv2 := returnValue2{IRet: 1, Cmd: "getmsg"}
		fmt.Fprintf(w, pub_tostring(rv2))

	}

}

// StringReplace -- replaces all occurences of rep with sub in src
func StringReplace(src, rep, sub string) (n string) {
	// make sure the src has the char we want to replace.
	if strings.Count(src, rep) > 0 {
		runes := src // convert to utf-8 runes.
		for i := 0; i < len(runes); i++ {
			l := string(runes[i]) // grab our rune and convert back to string.
			if l == rep {
				n += sub
			} else {
				n += l
			}
		}
		return n
	}
	return src
}
func say(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")
	msg := r.FormValue("msg")

	msg = StringReplace(msg, " ", "+")

	isVisitor := (uid == "")

	rv := returnValue{IRet: 0, Cmd: "say", Uid: uid, Tuid: tuid}

	user, ok := pub_getUserFromMap(uid)
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("say: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := pub_getUserFromMap(tuid)
	if ok {

	} else {
		tuser = &User{uid: tuid, session: "NA"}
		tuser.ch = make(chan string, CHAN_BUFFER_SIZE)

		pub_setUserFromMap(tuid, tuser)
		Trace("say: new to user ", tuid)
	}

	//msg = fmt.Sprintf("[msg]{act:say,uid:%s,tuid:%s,msg:%s,time:%s}", uid, tuid, msg, time.Now().String())
	//msg = fmt.Sprintf("[msg]%s say to %s:%s", uid, tuid, msg)
	tuser.MM = msg

	tuser.LastAccessTime = time.Now()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(2e9)
		timeout <- true
	}()

	rv.IRet = 1
	rv.Msg = msg

	select {
	case tuser.ch <- pub_tostring(rv):

		fmt.Fprintf(w, pub_tostring(rv))
	case <-timeout:
		rv.IRet = 0
		rv.Msg = "Timeout"
		fmt.Fprintf(w, pub_tostring(rv))

	}

	//tuser.ch <- pub_tostring(rv)
}

func pub_tostring(rv interface{}) string {
	b, _ := json.Marshal(rv)
	Trace(string(b))
	return string(b)
}

func log(w http.ResponseWriter, r *http.Request) {
	l := r.FormValue("level")
	iLevel, _ := strconv.ParseInt(l, 10, 32)
	SetLevel(int(iLevel))
	fmt.Fprintf(w, "level="+strconv.Itoa(Level()))
	return

}
func logon(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	pass := r.FormValue("pass")

	isVisitor := (uid == "")

	rv := returnValue{IRet: 0, Cmd: "logon", Uid: uid}

	err := userLogin(uid, pass)
	if err != nil {
		Info(err.Error())
		time.Sleep(2e9)
		rv.Msg = err.Error()
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	user, ok := pub_getUserFromMap(uid)
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("LOGON: new from user ", uid)
	}

	user.session, _ = pub_GenSession()
	user.isOnline = true

	cookie := http.Cookie{Name: "session", Value: user.session, Path: "/", HttpOnly: true}
	http.SetCookie(w, &cookie)

	//Trace("LOGIN SUCCESS: " + uid)

	rv.IRet = 1
	rv.Msg = "LOGIN SUCCESS"
	fmt.Fprintf(w, pub_tostring(rv))

	user.LastAccessTime = time.Now()

}

func signup(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	pass := r.FormValue("pass")

	rv := returnValue{IRet: 0, Cmd: "logon", Uid: uid}

	err := userSignup(uid, pass)
	if err != nil {
		Info(err.Error())
		time.Sleep(2e9)
		rv.Msg = err.Error()
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	rv.IRet = 1
	rv.Msg = "signup SUCCESS"
	fmt.Fprintf(w, pub_tostring(rv))

}
func logoff(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")

	isVisitor := (uid == "")

	rv := returnValue{IRet: 0, Cmd: "logoff", Uid: uid}

	user, ok := pub_getUserFromMap(uid)
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("LOGOFF: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	user.MM = "offline"
	user.session = "NA"
	user.isOnline = false

	rv.IRet = 1
	rv.Msg = "offline success"
	rvs := pub_tostring(rv)

	fmt.Fprintf(w, rvs)

	user.ch <- rvs

}

func joinroom(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	cid := r.FormValue("cid")

	isVisitor := (uid == "")
	if isVisitor {
		uid = pub_GetVid(r)
	}

	rv := returnValue{IRet: 0, Cmd: "joinroom", Uid: uid, Cid: cid}
	//var room *list.List

	user, ok := pub_getUserFromMap(uid)
	if ok {
	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("joinroom: new user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	room, ok := pub_getRoomFromMap(cid)
	if ok {

	} else {
		room = list.New()

		pub_setRoomFromMap(cid, room)
	}
	if !UserinList(room, user) {

		room.PushFront(user)
	}
	rv.IRet = 1
	rvs := pub_tostring(rv)

	for element := room.Front(); element != nil; element = element.Next() {
		value := element.Value.(*User)
		value.ch <- rvs

	}
	//runtime.Gosched()

	fmt.Fprintf(w, rvs)
}

func UserinList(room *list.List, user *User) bool {
	for element := room.Front(); element != nil; element = element.Next() {
		value := element.Value.(*User)
		if user == value {
			return true
		}
	}
	return false
}
func DeleteUserinList(room *list.List, user *User) bool {
	for element := room.Front(); element != nil; element = element.Next() {
		value := element.Value.(*User)
		if user == value {
			room.Remove(element)
			return true
		}
	}
	return false
}

func sayroom(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	cid := r.FormValue("cid")
	msg := r.FormValue("msg")

	msg = StringReplace(msg, " ", "+")

	isVisitor := (uid == "")
	if isVisitor {
		uid = pub_GetVid(r)
	}

	Trace("[" + msg + "]")

	rv := returnValue{IRet: 0, Cmd: "sayroom", Uid: uid, Cid: cid}

	room, ok := pub_getRoomFromMap(cid)
	if ok {

	} else {
		room = list.New()
		pub_setRoomFromMap(cid, room)
	}

	user, ok := pub_getUserFromMap(uid)
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("sayroom: new user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	if !UserinList(room, user) {
		room.PushFront(user)
	}
	rv.IRet = 1
	rv.Msg = msg

	rvs := pub_tostring(rv)

	for element := room.Front(); element != nil; element = element.Next() {
		value := element.Value.(*User)
		value.ch <- rvs

	}
	//runtime.Gosched()
	fmt.Fprintf(w, rvs)

}

func leftroom(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	cid := r.FormValue("cid")

	isVisitor := (uid == "")
	if isVisitor {
		uid = pub_GetVid(r)
	}

	rv := returnValue{IRet: 0, Cmd: "leftroom", Uid: uid, Cid: cid}

	user, ok := pub_getUserFromMap(uid)
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("leftroom: new user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	room, ok := pub_getRoomFromMap(cid)
	if ok {

	} else {
		room = list.New()
		pub_setRoomFromMap(cid, room)
	}

	DeleteUserinList(room, user)

	rv.IRet = 1
	rvs := pub_tostring(rv)

	for element := room.Front(); element != nil; element = element.Next() {
		value := element.Value.(*User)
		value.ch <- rvs
	}
	//runtime.Gosched()

	if room.Len() <= 0 {
		pub_delRoomFromMap(cid)

		Trace("room deleted  ", cid)
	}

	fmt.Fprintf(w, rvs)
}

func call(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")
	company := r.FormValue("company")
	skill := r.FormValue("skill")

	isVisitor := (uid == "")
	if isVisitor {

		uid = pub_GetVid(r)

	}
	rv := returnValue{IRet: 0, Cmd: "call", Uid: uid, Tuid: tuid}

	//  uid , tuid ,company ,skill
	//         x                   | Visitor Call User
	//                x            | Visitor Call company
	//                x        x   | Visitor Call Skill

	if company != "" {
		var err error
		var users [10]string
		if skill == "" {
			err, users = getUsersInCompany(company)
		} else {
			err, users = getUsersInCompanySkill(company, skill)
		}

		if err != nil {
			rv.IRet = 0
			rv.Msg = err.Error()
			fmt.Fprintf(w, pub_tostring(rv))
			return
		} else {
			//brocase to all users

			rv.IRet = 0
			rv.Msg = fmt.Sprint(len(users), users[0])
			fmt.Fprintf(w, pub_tostring(rv))
			return

		}

	}
	if tuid != "" {

		user, ok := pub_getUserFromMap(uid)
		if ok {

		} else {
			user = &User{uid: uid, session: "NA"}
			user.isVisitor = isVisitor
			user.ch = make(chan string, CHAN_BUFFER_SIZE)
			pub_setUserFromMap(uid, user)
			Trace("call: new a Visitor ", uid)
		}

		if !pub_VerifySession(user, r) {
			time.Sleep(5e9)
			rv.IRet = 0
			rv.Msg = "Please Login"
			fmt.Fprintf(w, pub_tostring(rv))
			return
		}

		tuser, ok := pub_getUserFromMap(tuid)
		if ok {

		} else {
			tuser = &User{uid: tuid, session: "NA"}
			tuser.ch = make(chan string, CHAN_BUFFER_SIZE)
			pub_setUserFromMap(tuid, tuser)
			Trace("call: new to user ", tuid)
		}

		rv.IRet = 1
		rvs := pub_tostring(rv)

		if isVisitor {

			cookie := http.Cookie{Name: "vid", Value: user.uid, Path: "/", HttpOnly: true}
			http.SetCookie(w, &cookie)
		}
		fmt.Fprintf(w, rvs)
		tuser.ch <- rvs
	}
}

func cancelcall(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")

	isVisitor := (uid == "")
	if isVisitor {
		uid = pub_GetVid(r)
	}

	rv := returnValue{IRet: 0, Cmd: "cancelcall", Uid: uid, Tuid: tuid}

	user, ok := pub_getUserFromMap(uid)
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := pub_getUserFromMap(tuid)
	if ok {

	} else {
		tuser = &User{uid: tuid, session: "NA"}
		tuser.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(tuid, tuser)
		Trace("SETMSG: new to user ", tuid)
	}

	rv.IRet = 1
	rvs := pub_tostring(rv)

	fmt.Fprintf(w, rvs)
	tuser.ch <- rvs

}

func acceptcall(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")

	isVisitor := (uid == "")

	//cid, _ := pub_GenSession()
	cid := uid + "_" + tuid

	rv := returnValue{IRet: 0, Cmd: "acceptcall", Uid: uid, Tuid: tuid}

	user, ok := pub_getUserFromMap(uid)
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := pub_getUserFromMap(tuid)
	if ok {

	} else {

		tuser = &User{uid: tuid, session: "NA"}

		tuser.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(tuid, tuser)
		Trace("SETMSG: new to user ", tuid)
	}

	room, ok := pub_getRoomFromMap(cid)
	if ok {

	} else {
		room = list.New()
		pub_setRoomFromMap(cid, room)
	}

	if !UserinList(room, user) {

		room.PushFront(user)
	}

	if !UserinList(room, tuser) {

		room.PushFront(tuser)
	}

	rv.Cid = cid
	rv.IRet = 1

	rvs := pub_tostring(rv)

	fmt.Fprintf(w, rvs)
	tuser.ch <- rvs
}
func ignorecall(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")

	isVisitor := (uid == "")

	rv := returnValue{IRet: 0, Cmd: "ignorecall", Uid: uid, Tuid: tuid}

	user, ok := pub_getUserFromMap(uid)
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := pub_getUserFromMap(tuid)
	if ok {

	} else {
		tuser = &User{uid: tuid, session: "NA"}
		tuser.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(tuid, tuser)
		Trace("SETMSG: new to user ", tuid)
	}

	rv.IRet = 1
	rvs := pub_tostring(rv)

	fmt.Fprintf(w, rvs)
	tuser.ch <- rvs

}

func getuserstatus(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")

	isVisitor := (uid == "")

	rv := returnValue{IRet: 0, Cmd: "getuserstatus", Uid: uid, Tuid: tuid}

	user, ok := pub_getUserFromMap(uid)
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, CHAN_BUFFER_SIZE)
		pub_setUserFromMap(uid, user)
		Trace("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := pub_getUserFromMap(tuid)
	if ok {
		if tuser.isOnline {
			rv.IRet = 1
			rv.Msg = "Online"
			fmt.Fprintf(w, pub_tostring(rv))
		} else {
			rv.IRet = 1
			rv.Msg = "Offline"
			fmt.Fprintf(w, pub_tostring(rv))
		}

	} else {
		rv.IRet = 1
		rv.Msg = "Offline"
		fmt.Fprintf(w, pub_tostring(rv))
	}

}

var isBreak bool

func main() {
	//UserDB = make(map[string]*User)
	//ChatRoom = make(map[string]*list.List)

	readOptions()
	loadDB()

	http.Handle("/html/", http.FileServer(http.Dir("")))

	http.HandleFunc("/", Handler)
	http.HandleFunc("/log", log)

	http.HandleFunc("/logon", logon)
	http.HandleFunc("/getmsg", getmsg)
	http.HandleFunc("/logoff", logoff)
	http.HandleFunc("/say", say)

	http.HandleFunc("/joinroom", joinroom)
	http.HandleFunc("/leftroom", leftroom)
	http.HandleFunc("/sayroom", sayroom)

	http.HandleFunc("/call", call)
	http.HandleFunc("/cancelcall", cancelcall)
	http.HandleFunc("/acceptcall", acceptcall)
	http.HandleFunc("/ignorecall", ignorecall)

	http.HandleFunc("/signup", signup)

	http.HandleFunc("/getuserstatus", getuserstatus)

	//http.ListenAndServe(":80", nil)
	isBreak = false
	go func() {
		for {
			time.Sleep(10e9)
			//Trace("Checking timeout...")
			UserDBLock.RWMutex.Lock()
			for _, user := range UserDBLock.m {
				//Trace("Checking " + user.uid)
				if time.Since(user.LastAccessTime) > 60*10e8 {

					Info("dropping " + user.uid)
					delete(UserDBLock.m, user.uid)
					//UserDB[user.uid] = nil
				}

			}
			UserDBLock.RWMutex.Unlock()

			if isBreak {
				break
			}

		}
	}()

	http.ListenAndServe(ids.Listen, nil)

}

func pub_GenSession() (string, error) {
	uuid := make([]byte, 8)
	n, err := rand.Read(uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	return hex.EncodeToString(uuid), nil
}

func pub_CheckPass(uid string, pass string) bool {
	result := false
	for _, id := range ids.Ids {
		if id.Id == uid {
			if pass == id.Password {
				result = true
			}
		}
	}
	return result
}
func pub_VerifySession(user *User, r *http.Request) bool {

	user.LastAccessTime = time.Now()

	if user.isVisitor {
		return true
	}

	//==========session ==========
	session := ""
	cookie, err := r.Cookie("session")
	if err == nil {
		session = cookie.Value
		//Trace(cookie.Value)
	}
	//==========session ==========

	if session != user.session {
		Trace("pub_VerifySession", user.uid, session, user.session)
		return false
	}

	return true

}

func pub_GetVid(r *http.Request) string {

	vid := ""

	//==========session ==========

	cookie, err := r.Cookie("vid")
	if err == nil {
		vid = cookie.Value
		Trace("Get vid from Cookie", vid)
	}
	//==========session ==========

	if vid == "" {
		vid, _ = pub_GenSession()
		Trace("pub_GenSession vid", vid)
	}

	return vid

}

//===========================
//  Read Options
//===========================

type idPassType struct {
	Id       string //captital is very import!!
	Password string
}

type idsType struct {
	Listen string
	Ids    []idPassType
}

type returnValue struct {
	IRet int
	Cmd  string
	Uid  string
	Tuid string
	Cid  string
	Msg  string
}

type returnValue2 struct {
	IRet int
	Cmd  string
}

var ids idsType

func readOptions() {
	file, e := ioutil.ReadFile("./httpserver.json")

	if e != nil {
		Error("File error: %v\n", e)
		os.Exit(1)
	}
	Info(string(file))

	json.Unmarshal(file, &ids)
	Info("Results:", ids)
}

//===========================

//	http.HandleFunc("/getmsg", getmsg)
//        1.logon 2.say  3.logoff if in same dept
//         4.JoinRoom 5.leftroom 6.sayroom
// 7.call 8.cancelcall )

//	http.HandleFunc("/logoff", logoff)
//	http.HandleFunc("/say", say)

//	http.HandleFunc("/joinroom", joinroom)
//	http.HandleFunc("/leftroom", leftroom)
//	http.HandleFunc("/sayroom", sayroom)

//	http.HandleFunc("/call", call) //say
//	http.HandleFunc("/cancelcall", cancelcall) //say
//	http.HandleFunc("/acceptcall", acceptcall) //joinroom say
//	http.HandleFunc("/invite", invite) //joinroom say

//http.HandleFunc("/get", getEnv)
//
//func getEnv(writer http.ResponseWriter, req *http.Request) {
//	env := os.Environ()
//	writer.Write([]byte("<h1>Envirment</h1><br>"))
//	for _, v := range env {
//		writer.Write([]byte(v + "<br>"))
//	}
//	writer.Write([]byte("<br>"))
//}
