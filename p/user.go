package p

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/asdine/storm"
	"time"
)

type User struct {
	Username     string `storm:"id"`
	Email        string
	Password     string
	Avatar       string
	Version      int64
	UsersVersion int64
}

func user(username string)(u *User,err error){
	u = new(User)
	err = db.One("Username", username, u)
	return
}

func authUser(u *User) (user *User, err error) {
	user = new(User)
	if u.Email != "" {
		err = db.One("Email", u.Email, user)
	} else if u.Username != "" {
		err = db.One("Username", u.Username, user)
	}

	if err != nil && err == storm.ErrNotFound {
		err = ErrorUserNotExist
	}
	if err == nil && user.Password != u.Password {
		err = ErrorForbidden
	}

	if !(err == nil && user.Password == u.Password) {
		user = nil
	}

	return
}

func registerUser(user *User) (err error) {
	if user.Username == "" || user.Email == "" || user.Password == "" {
		return ErrorEmptyField
	}

	u, err := authUser(user)

	if u != nil || err == ErrorForbidden {
		return ErrorUserExisted
	}

	user.Version =time.Now().Unix()

	err = db.Save(user)

	return
}

type follow struct {
	username string `storm:"id"`
}

func updateUser(user *User)(err error){
	user.Version =time.Now().Unix()

	err = db.Update(user)

	return
}

func (u *User) following()(users []string,err error){
	users=make([]*follow,0)
	node:=db.From("users", u.Username)
	err=node.All(users)
	return
}

func (u *User) follow(username string) (err error) {
	user := new(User)
	err = db.One("Username", username, user)
	if err != nil {
		if err == storm.ErrNotFound {
			return ErrorUserNotExist
		}
		return
	}

	tx, err := db.Begin(true)
	u.UsersVersion =time.Now().Unix()
	err=db.Save(u)
	if err!=nil {
		return
	}
	n:=db.From("users", u.Username)
	n = n.WithTransaction(tx)
	err=n.Save(&follow{username})
	if err!=nil{
		return
	}
	return tx.Commit()
}

func (u *User) unfollow(username string) (err error) {
	user := new(User)
	err = db.One("Username", username, user)
	if err != nil {
		if err == storm.ErrNotFound {
			return ErrorUserNotExist
		}
		return
	}

	tx, err := db.Begin(true)
	u.UsersVersion =time.Now().Unix()
	err=db.Save(u)
	if err!=nil {
		return
	}
	n:=db.From("users", u.Username)
	n = n.WithTransaction(tx)
	err=n.Drop(&follow{username})
	if err!=nil{
		return
	}
	return tx.Commit()
}

func isFollowing(follower,username string)bool{
	node:=db.From("users", follower)
	err:=node.One("username",username,new(follow))
	return err==nil
}

func (u *User) token(uuid string) *jwt.Token {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"UUID": uuid,
		"Username": u.Username,
		"Password":u.Password,
		"Email":u.Email,
	})
	return token
}



