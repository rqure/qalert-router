package main

import (
	"context"

	qdb "github.com/rqure/qdb/src"
	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/data"
	"github.com/rqure/qlib/pkg/data/notification"
	"github.com/rqure/qlib/pkg/data/query"
	"github.com/rqure/qlib/pkg/log"
)

type AlertWorker struct {
	store              data.Store
	isLeader           bool
	notificationTokens []data.NotificationToken
}

func NewAlertWorker(store data.Store) *AlertWorker {
	return &AlertWorker{
		db:                 db,
		isLeader:           false,
		notificationTokens: []data.NotificationToken{},
	}
}

func (w *AlertWorker) OnBecameLeader(context.Context) {
	w.isLeader = true

	w.notificationTokens = append(w.notificationTokens, w.store.Notify(
ctx,
notification.NewConfig().
SetEntityType(        "AlertController").
SetFieldName(        "SendTrigger"),
	}, notification.NewCallback(w.ProcessNotification)))
}

func (w *AlertWorker) OnLostLeadership(context.Context) {
	w.isLeader = false

	for _, token := range w.notificationTokens {
		token.Unbind()
	}

	w.notificationTokens = []data.NotificationToken{}
}

func (w *AlertWorker) Init(context.Context, app.Handle) {

}

func (w *AlertWorker) Deinit(context.Context) {

}

func (w *AlertWorker) DoWork(context.Context) {

}

func (w *AlertWorker) ProcessNotification(ctx context.Context, notification data.Notification) {
	if !w.isLeader {
		return
	}

	log.Info("Received notification: %v", notification)

	applicationName := notification.GetContext(0).GetValue().GetString()
	description := notification.GetContext(1).GetValue().GetString()
	ttsAlert := notification.GetContext(2).GetValue().GetBool()
	emailAlert := notification.GetContext(3).GetValue().GetBool()

	if ttsAlert {
		log.Info("Sending TTS alert: %v", description)

		controllers := query.New(w.store).Find(qdb.SearchCriteria{
			EntityType: "AudioController",
			Conditions: []qdb.FieldConditionEval{},
		})

		for _, controller := range controllers {
			controller.GetField("TextToSpeech").WriteString(ctx, description)
		}
	}

	if emailAlert {
		log.Info("Sending email alert: %v", description)

		controllers := query.New(w.store).Find(qdb.SearchCriteria{
			EntityType: "SmtpController",
			Conditions: []qdb.FieldConditionEval{},
		})

		for _, controller := range controllers {
			// Needs to be written as an atomic bulk operation so notifications don't get mingled together
			w.store.Write([]*qdb.DatabaseRequest{
				{
					Id:    controller.GetId(),
					Field: "Subject",
					Value: qdb.NewStringValue("Alert from " + applicationName + " service"),
				},
				{
					Id:    controller.GetId(),
					Field: "Body",
					Value: qdb.NewStringValue(description),
				},
				{
					Id:    controller.GetId(),
					Field: "SendTrigger",
					Value: qdb.NewIntValue(0),
				},
			})
		}
	}
}
