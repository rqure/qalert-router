package main

import (
	"os"

	qdb "github.com/rqure/qdb/src"
	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/app/workers"
	"github.com/rqure/qlib/pkg/data/store"
)

func getDatabaseAddress() string {
	addr := os.Getenv("Q_ADDR")
	if addr == "" {
		addr = "ws://webgateway:20000/ws"
	}

	return addr
}

func main() {
	db := store.NewWeb(store.WebConfig{
		Address: getDatabaseAddress(),
	})

	storeWorker := workers.NewStore(db)
	leadershipWorker := workers.NewLeadership(db)
	alertWorker := NewAlertWorker(db)
	schemaValidator := leadershipWorker.GetEntityFieldValidator()

	schemaValidator.RegisterEntityFields("Root", "SchemaUpdateTrigger")
	schemaValidator.RegisterEntityFields("AlertController", "ApplicationName", "Description", "TTSAlert", "EmailAlert", "SendTrigger")

	storeWorker.Connected.Connect(leadershipWorker.OnStoreConnected)
	storeWorker.Disconnected.Connect(leadershipWorker.OnStoreDisconnected)

	leadershipWorker.BecameLeader().Connect(alertWorker.OnBecameLeader)
	leadershipWorker.LosingLeadership().Connect(alertWorker.OnLostLeadership)

	// Create a new application configuration
	config := qdb.ApplicationConfig{
		Name: "alert",
		Workers: []qdb.IWorker{
			storeWorker,
			leadershipWorker,
			alertWorker,
		},
	}

	app := app.NewApplication(config)

	app.Execute()
}
