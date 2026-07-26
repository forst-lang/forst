package main

type Config struct {
	Host  string `json:"host"`
	Port  int    `json:"port,omitempty"`
	plain int
}

func main() {
	c := Config{Host: "localhost", Port: 8080, plain: 1}
	println(c.Host, c.Port, c.plain)
}
