package dgraph

import (
	"context"
	"log"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Dg is the global Dgraph client instance
var Dg *dgo.Dgraph

// InitDgraph initializes the connection to Dgraph
func InitDgraph(address string) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Dgraph: %v", err)
	}

	dc := api.NewDgraphClient(conn)
	Dg = dgo.NewDgraphClient(dc)

	// Drop all existing data to start fresh
	op := &api.Operation{
		DropAll: true,
	}

	if err := Dg.Alter(context.Background(), op); err != nil {
		log.Printf("Failed to drop all data (may be first run): %v", err)
	}

	// Now set the schema
	op = &api.Operation{
		Schema: `
			id: string @index(exact) .
			name: string .
			clock: string .
			depth: int .
			value: string .
			key: string .
			node: string .
			parent: [uid] .
			type Event {
				id
				name
				clock
				depth
				parent
				value
				key
				node
			}
		`,
	}

	if err := Dg.Alter(context.Background(), op); err != nil {
		log.Fatalf("Failed to set schema: %v", err)
	}

	log.Println("Connected to Dgraph, cleared all data, and schema set successfully")
}
