package consul

import (
	"fmt"
	"testing"

	"github.com/pojol/braid/mock"
)

func TestServicesList(t *testing.T) {

	_, err := ServicesList(mock.ConsulAddr)
	if err != nil {
		t.Error(err)
	}

}

func TestGetBoxServices(t *testing.T) {

	lst, err := GetCatalogServices(mock.ConsulAddr, "redis")
	if err != nil {
		t.Error(err)
	}

	fmt.Println(lst)

}