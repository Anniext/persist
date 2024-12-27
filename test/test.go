package test

import (
	"fmt"
	"persist/data"
	"testing"
)

func TestPersist(t *testing.T) {
	args := data.GItemLocalManager.GetAll()
	fmt.Println(args)
}
