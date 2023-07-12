package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/provider"
	lts "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/lts/v2"
)

var stdout = log.New(os.Stdout, "", 0)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	auth, err := provider.BasicCredentialEnvProvider().GetCredentials()
	if err != nil {
		log.Fatalf("[FATAL] fail instantiating credentials: %v\nSee https://github.com/huaweicloud/huaweicloud-sdk-go-v3#241-environment-variables-top.", err)
	}
	ltsc := lts.NewLtsClient(lts.LtsClientBuilder().
		WithCredential(auth).
		WithRegion(regionFromEnv()).
		Build())

	errc := make(chan error)
	go func() {
		mgr := LogstreamManager{
			client:        ltsc,
			streams:       map[LogstreamID]*Logstream{},
			queue:         make(chan *Logstream),
			maxEndFromNow: maxEndFromNow,
			maxFetchRange: maxFetchRange,
			minFetchRange: minFetchRange,
		}
		errc <- mgr.Start(ctx, streamRoutine)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigs:
		cancel()

	case err := <-errc:
		if err != nil {
			log.Fatalf("[FATAL] fail to get initial log groups and streams: %v\n", err)
		}
	}
}
