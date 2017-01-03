package main

import (
	"fmt"
)
func boo(s string)(n string){
	fmt.Println(s,n)
	return
}
func foo(s string)(n string){
	n=s
	boo(s)
	return
}

func main() {
	foo("ok")
}


