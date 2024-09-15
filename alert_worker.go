package main

import (
	qdb "github.com/rqure/qdb/src"
)

type AlertWorker struct {
	db                 qdb.IDatabase
	isLeader           bool
	notificationTokens []qdb.INotificationToken
}

func NewAlertWorker(db qdb.IDatabase) *AlertWorker {
	return &AlertWorker{
		db:                 db,
		isLeader:           false,
		notificationTokens: []qdb.INotificationToken{},
	}
}

func (w *AlertWorker) OnBecameLeader() {
	w.isLeader = true

	w.notificationTokens = append(w.notificationTokens, w.db.Notify(&qdb.DatabaseNotificationConfig{
		Type:          "AlertController",
		Field:         "SendTrigger",
		ContextFields: []string{"ApplicationName", "Description", "TTSAlert", "EmailAlert"},
	}, qdb.NewNotificationCallback(w.ProcessNotification)))
}

func (w *AlertWorker) OnLostLeadership() {
	w.isLeader = false

	for _, token := range w.notificationTokens {
		token.Unbind()
	}

	w.notificationTokens = []qdb.INotificationToken{}
}

func (w *AlertWorker) Init() {

}

func (w *AlertWorker) Deinit() {

}

func (w *AlertWorker) DoWork() {

}

func (w *AlertWorker) ProcessNotification(notification *qdb.DatabaseNotification) {
	if !w.isLeader {
		return
	}

	qdb.Info("[AlertWorker::ProcessNotification] Received notification: %v", notification)

	applicationName := qdb.ValueCast[*qdb.String](notification.Context[0].Value).Raw
	description := qdb.ValueCast[*qdb.String](notification.Context[1].Value).Raw
	ttsAlert := qdb.ValueCast[*qdb.Bool](notification.Context[2].Value).Raw
	emailAlert := qdb.ValueCast[*qdb.Bool](notification.Context[3].Value).Raw

	if ttsAlert {
		qdb.Info("[AlertWorker::ProcessNotification] Sending TTS alert: %v", description)

		controllers := qdb.NewEntityFinder(w.db).Find(qdb.SearchCriteria{
			EntityType: "AudioController",
			Conditions: []qdb.FieldConditionEval{},
		})

		for _, controller := range controllers {
			controller.GetField("TextToSpeech").PushString(description)
		}
	}

	if emailAlert {
		qdb.Info("[AlertWorker::ProcessNotification] Sending email alert: %v", description)

		controllers := qdb.NewEntityFinder(w.db).Find(qdb.SearchCriteria{
			EntityType: "SmtpController",
			Conditions: []qdb.FieldConditionEval{},
		})

		for _, controller := range controllers {
			// Needs to be written as an atomic bulk operation so notifications don't get mingled together
			w.db.Write([]*qdb.DatabaseRequest{
				{
					Id:    controller.GetId(),
					Field: "Subject",
					Value: qdb.NewStringValue("Alert from '" + applicationName + "' service"),
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
