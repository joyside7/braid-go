package consul

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pojol/braid-go/mock"
)

func TestSession(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(1000)

	sessionName := "test" + strconv.Itoa(r)

	id, err := CreateSession(mock.ConsulAddr, sessionName)
	assert.Equal(t, err, nil, err.Error())
	CreateSession("xxx", sessionName)

	err = RefushSession(mock.ConsulAddr, id)
	assert.Equal(t, err, nil, err.Error())
	RefushSession("xxx", id)

	ok, err := AcquireLock(mock.ConsulAddr, sessionName, id)
	assert.Equal(t, ok, true, err.Error())
	assert.Equal(t, err, nil, err.Error())
	AcquireLock("xxx", sessionName, id)

	err = ReleaseLock(mock.ConsulAddr, sessionName, id)
	assert.Equal(t, err, nil, err.Error())
	ReleaseLock("xxx", sessionName, id)

	ok, err = DeleteSession(mock.ConsulAddr, id)
	assert.Equal(t, ok, true, err.Error())
	assert.Equal(t, err, nil, err.Error())
	DeleteSession("xxx", id)
}
