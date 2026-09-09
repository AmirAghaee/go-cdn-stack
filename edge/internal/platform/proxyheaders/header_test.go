package proxyheaders

import (
	"reflect"
	"testing"
)

func TestEndToEnd(t *testing.T) {
	source := map[string][]string{
		"connection": {" keep-alive, X-Private", "x-other, Upgrade"},
		"X-Private":  {"secret"}, "x-other": {"secret"},
		"Keep-Alive": {"timeout=5"}, "Proxy-Connection": {"close"},
		"Proxy-Authenticate": {"secret"}, "Proxy-Authorization": {"secret"},
		"TE": {"trailers"}, "Trailer": {"X-Final"}, "Transfer-Encoding": {"chunked"}, "Upgrade": {"websocket"},
		"Content-Type": {"image/png"}, "X-Origin": {"one", "two"},
	}
	expected := map[string][]string{"Content-Type": {"image/png"}, "X-Origin": {"one", "two"}}
	actual := EndToEnd(source)
	if !reflect.DeepEqual(map[string][]string(actual), expected) {
		t.Fatalf("headers = %v", actual)
	}
	actual["X-Origin"][0] = "changed"
	if source["X-Origin"][0] != "one" || len(source["connection"]) != 2 {
		t.Fatal("mutated source")
	}
}
