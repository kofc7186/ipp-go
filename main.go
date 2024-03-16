package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"

	flag "github.com/spf13/pflag"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

var ipFlag net.IP
var printerURL url.URL

func init() {
	flag.IPVar(&ipFlag, "ip", net.ParseIP("192.168.2.246"), "ip address of printer")
	flag.Parse()
}

func main() {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, "fishfry2021", option.WithCredentialsFile("creds.json"))
	if err != nil {
		log.Fatal(err)
	}

	printerURL.Scheme = "ipp"
	printerURL.Host = ipFlag.String()

	slog.Info("subscribing to print_queue", "printerIP", printerURL.String())
	sub := client.Subscription("print_queue")
	sub.ReceiveSettings.MaxOutstandingMessages = 1
	if err = sub.Receive(ctx, ReceivePubsubMessage); err != nil {
		log.Println(err)
	}
}

func ReceivePubsubMessage(ctx context.Context, m *pubsub.Message) {
	slog.InfoContext(ctx, "received document to print", "attributes", m.Attributes)

	tempFile, err := os.CreateTemp("", m.ID)
	if err != nil {
		slog.Error("error creating temp file", "err", err)
		m.Nack()
		return
	}
	pdfBytes := m.Data
	if b64DecodedBytes, err := base64.StdEncoding.DecodeString(string(m.Data)); err == nil {
		pdfBytes = b64DecodedBytes
	}
	if _, err := io.Copy(tempFile, bytes.NewReader(pdfBytes)); err != nil {
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
	cmd := exec.Command("./cups/tools/ipptool-static", "-f", tempFile.Name(), printerURL.String(), "./print-job-and-wait.test")
	if err := cmd.Run(); err != nil {
		slog.ErrorContext(ctx, "error invoking ipptool", "err", err, "son", m.Attributes["son"])
		m.Nack()
		return
	}
	slog.Info("printed order successfully", "son", m.Attributes["son"])
	m.Ack()
}
