package p

import "errors"

var (
	ErrorForbidden=errors.New("forbidden")
	ErrorUserExisted=errors.New("user existed")
	ErrorUserNotExist=errors.New("user not exist")
	ErrorEmptyField=errors.New("empty field")
	ErrorCheckUserInfo=errors.New("check userInfo error")
	ErrUpdateUserInfoFailed=errors.New("update userInfo failed")
)

