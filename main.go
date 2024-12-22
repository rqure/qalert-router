package main

import (
	"os"

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
	s := store.NewWeb(store.WebConfig{
		Address: getDatabaseAddress(),
	})

	storeWorker := workers.NewStore(s)
	leadershipWorker := workers.NewLeadership(s)
	alertWorker := NewAlertWorker(s)
	schemaValidator := leadershipWorker.GetEntityFieldValidator()

	schemaValidator.RegisterEntityFields("Root", "SchemaUpdateTrigger")
	schemaValidator.RegisterEntityFields("AlertController", "ApplicationName", "Description", "TTSAlert", "EmailAlert", "SendTrigger")

	storeWorker.Connected.Connect(leadershipWorker.OnStoreConnected)
	storeWorker.Disconnected.Connect(leadershipWorker.OnStoreDisconnected)

	leadershipWorker.BecameLeader().Connect(alertWorker.OnBecameLeader)
	leadershipWorker.LosingLeadership().Connect(alertWorker.OnLostLeadership)

	a := app.NewApplication("alert")
	a.AddWorker(storeWorker)
	a.AddWorker(leadershipWorker)
	a.AddWorker(alertWorker)
	a.Execute()
}
