package controller

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

func produceMessage(kp KafkaProducer, message string, topic string) error {
	errCh := make(chan error)
	record := &kgo.Record{Topic: topic, Value: []byte(message)}
	kp.Produce(context.TODO(), record, func(_ *kgo.Record, err error) {
		if err != nil {
			fmt.Printf("record had a produce error: %v\n", err)
		}
		errCh <- err
	})
	err := <-errCh

	return err
}
