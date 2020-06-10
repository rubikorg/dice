package postgres

import (
	"testing"

	"github.com/rubikorg/dice"
)

func TestOpen(t *testing.T) {
	wrongURI := dice.ConnectURI{
		Database: "hello",
		Password: "world",
	}
	db, _ := Connect(wrongURI)
	err := db.Ping()
	if err == nil {
		t.Error("did not error when passed wrong dice.ConnectUri{}")
	}
}
