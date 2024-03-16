package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

var IPs = []string{"192.168.2.246"} //, "192.168.2.145"}

func main() {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, "fishfry2021", option.WithCredentialsFile("creds.json"))
	if err != nil {
		log.Fatal(err)
	}

	sub := client.Subscription("print_queue")
	sub.ReceiveSettings.MaxOutstandingMessages = 1
	if err = sub.Receive(ctx, ReceivePubsubMessage); err != nil {
		log.Println(err)
	}
}

func ReceivePubsubMessage(ctx context.Context, m *pubsub.Message) {
	slog.InfoContext(ctx, "received message", "attributes", m.Attributes)

	tempFile, err := os.CreateTemp("", m.ID)
	if err != nil {
		slog.Error("error creating temp file", "err", err)
		m.Nack()
		return
	}
	dec := bytes.NewReader(m.Data)
	//dec := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(m.Data))
	if _, err := io.Copy(tempFile, dec); err != nil {
		slog.Error("error writing PDF to temp file", "err", err)
		m.Nack()
		return
	}
	if err := tempFile.Sync(); err != nil {
		slog.Error("error syncing temp file", "err", err)
		m.Nack()
		return
	}
	if err := tempFile.Close(); err != nil {
		slog.Error("error closing temp file", "err", err)
		m.Nack()
		return
	}
	defer os.Remove(tempFile.Name())

	//cmd := exec.Command("/Users/bcallaway/git/kofc7186/ipp-go/cups/tools/ipptool-static", "-f", tempFile.Name(), "ipp://192.168.2.246", "./print-job-and-wait.test")
	cmd := exec.Command("./cups/tools/ipptool-static", "-f", tempFile.Name(), "ipp://192.168.2.246", "./print-job-and-wait.test")
	if err := cmd.Run(); err != nil {
		slog.ErrorContext(ctx, "error invoking ipptool", "err", err)
		m.Nack()
		return
	}
	slog.Info("printed order successfully")
	m.Ack()
}
