package p

import (
	"io/ioutil"
	"net/http"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"log"
	"time"
)

var RSA_KEY = func() []byte {
	key, e := ioutil.ReadFile("bino")
	if e != nil {
		panic(e.Error())
	}
	return key
}()

var RSA_PUB = func() []byte {
	key, e := ioutil.ReadFile("bino.pub")
	if e != nil {
		panic(e.Error())
	}
	return key
}()

func keyFunc(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
	}
	var user *User
	if _, ok := token.Claims.(jwt.MapClaims); !ok {
		return nil, fmt.Errorf("Unexpected claim type: %v", token.Claims)
	} else {
		claims := token.Claims.(jwt.MapClaims)
		u := &User{
			Username :claims["Username"].(string),
			Email    :claims["Email"].(string),
			Password :claims["Password"].(string),
		}
		if user, err := authUser(u); user != nil && err != nil {
			return nil, fmt.Errorf("auth failed: %v", auth)
		}
	}
	return user, nil
}

func authenticate(w http.ResponseWriter,r *http.Request) (ok bool, token *jwt.Token) {
	if r.RequestURI != "/auth" && r.RequestURI != "/register" {
		tokenCookie, err := r.Cookie("Authorization")
		ok = err == nil
		if !ok {
			return
		}
		token, err := jwt.Parse(tokenCookie.Value, keyFunc)
		ok = err != nil && token.Valid
	}
	return
}

func checkToken(tokenstr string) bool {
	token, err := jwt.Parse(tokenstr, keyFunc)
	return err != nil && token.Valid
}

func validUUID(uuid string) bool {
	return true
}

type Authentication struct {
	UUID     string
	Username string
	Email    string
	Password string
}

func auth(w http.ResponseWriter,r *http.Request) {
	auth := new(Authentication)

	err := json.NewDecoder(r.Body).Decode(auth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	uuid := auth.UUID
	if !validUUID(uuid) {
		http.Error(w, fmt.Sprint("not acceptable uuid", uuid),
			http.StatusBadRequest)
		return
	}
	user := &User{
		Username :auth.Username,
		Email    :auth.Email,
		Password :auth.Password,
	}
	_, err = authUser(user)

	if err == ErrorForbidden {
		http.Error(w, ErrorForbidden.Error(),
			http.StatusForbidden)
		return
	}

	if err == ErrorUserNotExist {
		http.Error(w, ErrorUserNotExist.Error(),
			http.StatusNotAcceptable)
		return
	}

	if err != nil {
		http.Error(w, err.Error(),
			http.StatusInternalServerError)
		return
	}

	tokenstr,err:=user.token(uuid).SignedString(RSA_KEY)
	if err!=nil{
		http.Error(w, err.Error(),
			http.StatusInternalServerError)
		return
	}

	cookie := tokenCookie(tokenstr)
	http.SetCookie(w, cookie)
	w.WriteHeader(http.StatusOK)
}

type Registration struct {
	UUID     string
	Email    string
	Password string
	Username string
}

func register(w http.ResponseWriter,r *http.Request) {
	reg := new(Registration)
	err := json.NewDecoder(r.Body).Decode(reg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	uuid := reg.UUID
	if !validUUID(uuid) {
		http.Error(w, fmt.Sprint("not acceptable uuid", uuid),
			http.StatusBadRequest)
		return
	}
	user := &User{
		Username :reg.Username,
		Email    :reg.Email,
		Password :reg.Password,
	}
	err = registerUser(user)
	if err != nil {
		if err == ErrorEmptyField {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err == ErrorUserExisted {
			http.Error(w, ErrorUserExisted.Error(),
				http.StatusNotAcceptable)
			return
		}
		log.Println(err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	tokenstr,err:=user.token(uuid).SignedString(RSA_KEY)
	if err!=nil{
		http.Error(w, err.Error(),
			http.StatusInternalServerError)
		return
	}
	http.SetCookie(w,tokenCookie(tokenstr))
	w.WriteHeader(http.StatusCreated)

}

var tokenTimeDur=func()time.Duration{
	dur,err:=time.ParseDuration("3600s")
	if err!=nil {
		log.Fatalln(err)
	}
	return dur
}()

func tokenCookie(token string)*http.Cookie{
	return &http.Cookie{
		Name:"Authorization",
		Value:token,
		Path:"/",
		MaxAge:3600,
		Secure:false,
		HttpOnly:true,
	}
}
