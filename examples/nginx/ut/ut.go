package main

import (
	"fmt"
	"os/user"
)

func main() {
	os_user, err := user.Lookup("root")
	if err != nil {
		fmt.Errorf("Error looking up user: '%s'", err)
	}
	fmt.Println("os_user = ", os_user)
}
