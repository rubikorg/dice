package mgoconn

import (
	"context"
	"fmt"

	"github.com/rubikorg/dice"
	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/mongo/options"
)

func Connect(cred dice.ConnectURI) (*mongo.Database, error) {
	link := fmt.Sprintf("mongodb://%s:%d", cred.Host, cred.Port)
	client, err := mongo.NewClient(options.Client().ApplyURI(link))
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return client.Database(cred.Database), nil
}
