package queue

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
)

const (
	messageVisibilityTimeout = 60 // seconds
	messageHeartbeat         = 10 * time.Second
)

type queue struct {
	log *logrus.Entry
	q   *storage.Queue
}

// Queue represents a queue
type Queue interface {
	Get() (Message, error)
	Put(string) error
}

type message struct {
	log  *logrus.Entry
	m    *storage.Message
	stop chan struct{}
	done chan struct{}
}

// Message represents a message
type Message interface {
	ID() string
	Done(error) error
	DequeueCount() int
}

// NewQueue returns a new queue
func NewQueue(log *logrus.Entry, storageAccount, storageKey, queueName string) (Queue, error) {
	cli, err := storage.NewClient(storageAccount, storageKey, azure.PublicCloud.StorageEndpointSuffix, storage.DefaultAPIVersion, true)
	if err != nil {
		return nil, err
	}

	qsc := cli.GetQueueService()

	q := qsc.GetQueueReference(queueName)

	exists, err := q.Exists()
	if err != nil {
		return nil, err
	}

	if !exists {
		err = q.Create(nil) // can't do this via ARM template, unfortunately
		if err != nil {
			return nil, err
		}
	}

	return &queue{log: log, q: q}, nil
}

func (q *queue) Get() (Message, error) {
	ms, err := q.q.GetMessages(&storage.GetMessagesOptions{NumOfMessages: 1, VisibilityTimeout: messageVisibilityTimeout})
	if err != nil {
		return nil, err
	}

	switch {
	case len(ms) > 1:
		return nil, fmt.Errorf("read %d documents, expected <= 1", len(ms))
	case len(ms) == 0:
		return nil, nil
	}

	return q.newMessage(&ms[0]), nil
}

func (q *queue) Put(id string) error {
	m := &storage.Message{
		Queue: q.q,
		Text:  id,
	}

	return m.Put(&storage.PutMessageOptions{MessageTTL: -1})
}

func (q *queue) newMessage(sm *storage.Message) Message {
	m := &message{
		log:  q.log,
		m:    sm,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	go m.heartbeat()

	return m
}

func (m *message) heartbeat() {
	defer close(m.done)

	t := time.NewTicker(messageHeartbeat)
	defer t.Stop()

	for {
		err := m.m.Update(&storage.UpdateMessageOptions{VisibilityTimeout: messageVisibilityTimeout})
		if err != nil {
			m.log.Error(err)
		}

		select {
		case <-t.C:
		case <-m.stop:
			return
		}
	}
}

func (m *message) ID() string {
	return m.m.Text
}

func (m *message) Done(err error) error {
	close(m.stop)
	<-m.done

	if err != nil {
		return err
	}

	return m.m.Delete(nil)
}

func (m *message) DequeueCount() int {
	return m.m.DequeueCount
}
