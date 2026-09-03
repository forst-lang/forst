package main

import "net/url"

func main() {
	u := &url.URL{}
	println(u.Path)
}
