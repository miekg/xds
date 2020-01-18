package main

import "fmt"

const Version = "0.0.1"

func version(_ *Client, args []string) error {
	fmt.Println(Version)
	return nil
}
