package salesforce

import (
	"crm-middleware/api"
	"crm-middleware/httputil"
	"encoding/json"
	"net/http"
	"time"

	"log/slog"
)

type CreateTaskData struct {
	Name                  string `json:"name"`
	ConversationID        string `json:"conversation_id"`
	TenantCode            string `json:"tenant_code"`
	Extension             string `json:"ext"`
	CustomerID            string `json:"customer_id"`
	Answered              int    `json:"answered"`
	Duration              int64  `json:"duration"`
	Subject               string `json:"subject"`
	Description           string `json:"description"`
	Channel               string `json:"channel"`
	ConversationFinished  bool   `json:"conversation_finished"`
	ConversationStartTime int64  `json:"conversation_start_time"`
	ObjectType            string `json:"object_type"`
	ObjectID              string `json:"object_id"`
}

type TaskData struct {
	Subject               string `json:"Subject"`
	Status                string `json:"Status"`
	ActivityDate          string `json:"ActivityDate"`
	Description           string `json:"Description,omitempty"`
	CallDurationInSeconds int64  `json:"CallDurationInSeconds,omitempty"`
	CallDisposition       string `json:"CallDisposition,omitempty"`
	TaskSubtype           string `json:"TaskSubtype,omitempty"`
	WhatID                string `json:"WhatId,omitempty"`
	WhoID                 string `json:"WhoId,omitempty"`
}

func (salesforce *salesforce) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	claims := api.GetClaims(r)

	var req CreateTaskData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, "failed to decode task request", http.StatusBadRequest,
			httputil.SlogError(err),
			claims.SlogAttr(),
		)
		return
	}

	if req.Subject == "" {
		httputil.Error(w, "subject is required", http.StatusBadRequest, claims.SlogAttr())
		return
	}
	if req.ObjectID == "" {
		httputil.Error(w, "object_id is required", http.StatusBadRequest, claims.SlogAttr())
		return
	}
	if _, ok := salesforce.Objects[req.ObjectType]; !ok {
		httputil.Error(w, "unsupported object type: "+req.ObjectType, http.StatusBadRequest, claims.SlogAttr())
		return
	}

	var subtype = "Task"
	switch req.Channel {
	case "email":
		subtype = "Email"
	case "voice":
		subtype = "Call"
	}

	task := &TaskData{
		Subject:               req.Subject,
		Description:           req.Description,
		CallDurationInSeconds: req.Duration,
		Status:                "Completed",
		ActivityDate:          time.Now().Format("2006-01-02"),
		TaskSubtype:           subtype,
		CallDisposition:       "Unanswered",
	}

	if req.Answered > 0 {
		task.CallDisposition = "Answered"
	}

	if req.ObjectType == Account {
		task.WhatID = req.ObjectID
	} else {
		task.WhoID = req.ObjectID
	}

	data, err := json.Marshal(task)
	if err != nil {
		httputil.Error(w, "failed to marshal task data", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
		)
		return
	}

	if _, err := salesforce.Request(claims.ClientID, claims.UserID, http.MethodPost, servicesDataEndpoint("sobjects", Task, "", nil), data); err != nil {
		httputil.Error(w, "failed to create salesforce task", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
		)
		return
	}

	slog.Debug("salesforce task created", claims.SlogAttr())
	w.WriteHeader(http.StatusCreated)
}
