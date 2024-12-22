package main

import (
	"context"

	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/data"
	"github.com/rqure/qlib/pkg/data/binding"
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
		store:              store,
		isLeader:           false,
		notificationTokens: []data.NotificationToken{},
	}
}

func (w *AlertWorker) OnBecameLeader(ctx context.Context) {
	w.isLeader = true

	w.notificationTokens = append(w.notificationTokens, w.store.Notify(
		ctx,
		notification.NewConfig().
			SetEntityType("AlertController").
			SetFieldName("SendTrigger"),
		notification.NewCallback(w.ProcessNotification)))
}

func (w *AlertWorker) OnLostLeadership(ctx context.Context) {
	w.isLeader = false

	for _, token := range w.notificationTokens {
		token.Unbind(ctx)
	}

	w.notificationTokens = []data.NotificationToken{}
}

func (w *AlertWorker) Init(context.Context, app.Handle) {

}

func (w *AlertWorker) Deinit(context.Context) {

}

func (w *AlertWorker) DoWork(context.Context) {

}

func (w *AlertWorker) ProcessNotification(ctx context.Context, n data.Notification) {
	if !w.isLeader {
		return
	}

	log.Info("Received notification: %v", n)

	applicationName := n.GetContext(0).GetValue().GetString()
	description := n.GetContext(1).GetValue().GetString()
	ttsAlert := n.GetContext(2).GetValue().GetBool()
	emailAlert := n.GetContext(3).GetValue().GetBool()

	if ttsAlert {
		log.Info("Sending TTS alert: %v", description)

		controllers := query.New(w.store).
			ForType("AudioController").
			Execute(ctx)

		for _, controller := range controllers {
			controller.GetField("TextToSpeech").WriteString(ctx, description)
		}
	}

	if emailAlert {
		log.Info("Sending email alert: %v", description)

		multi := binding.NewMulti(w.store)
		controllers := query.New(multi).
			ForType("SmtpController").
			Execute(ctx)

		for _, controller := range controllers {
			// Needs to be written as an atomic bulk operation so notifications don't get mingled together
			controller.GetField("Subject").WriteString(ctx, "Alert from "+applicationName+" service")
			controller.GetField("Body").WriteString(ctx, description)
			controller.GetField("SendTrigger").WriteInt(ctx)
		}

		multi.Commit(ctx)
	}
}
