package ibapi

import (
	"fmt"
	"testing"
)

func TestConnection(t *testing.T) {
	fmt.Println("connection testing!")
	conn := &IbConnection{}
	conn.connect("127.0.0.1", 4002)
	buf := make([]byte, 4096)
	_, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	fmt.Println(string(buf))
	conn.disconnect()
}
