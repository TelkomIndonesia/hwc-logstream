package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/lts/v2/model"
)

type LogstreamManagerClient interface {
	LogstreamClient

	ListLogGroups(request *model.ListLogGroupsRequest) (*model.ListLogGroupsResponse, error)
	ListLogStream(request *model.ListLogStreamRequest) (*model.ListLogStreamResponse, error)
}

type LogstreamManager struct {
	client  LogstreamManagerClient
	streams map[LogstreamID]*Logstream
	queue   chan *Logstream

	maxEndFromNow time.Duration
	maxFetchRange time.Duration
	minFetchRange time.Duration
	maxLag        time.Duration
}

func (m LogstreamManager) Start(ctx context.Context, routine int) (err error) {
	if err = m.SyncStreamList(ctx); err != nil {
		return fmt.Errorf("fail to initialy sync log group and streams: %w", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(routine + 1)
	defer wg.Wait()

	go func() {
		defer wg.Done()

		for {
			select {
			case <-time.After(time.Minute):
			case <-ctx.Done():
				return
			}

			err := m.SyncStreamList(ctx)
			if err != nil {
				log.Fatalf("[WARN] fail to sync log groups and streams: %v\n", err)
			}
		}
	}()

	for i := 0; i < routine; i++ {
		go func() {
			defer wg.Done()

			var s *Logstream
			for {
				select {
				case s = <-m.queue:
				case <-ctx.Done():
					return
				}

				if _, ok := m.streams[s.ID()]; !ok {
					continue
				}

				s.SkiptoCatchUp(m.maxLag)

				end := time.Now().Add(-m.maxEndFromNow)
				if e := s.start.Add(m.maxFetchRange); e.Before(end) {
					end = e
				}

				err := s.Stream(ctx, end)
				if err != nil {
					log.Printf("[WARN] got error when streaming: %v\n", err)
				}
				go m.Queue(ctx, s)
			}
		}()
	}

	return
}

func (m LogstreamManager) SyncStreamList(ctx context.Context) (err error) {
	groups, err := m.client.ListLogGroups(&model.ListLogGroupsRequest{})
	if err != nil {
		return fmt.Errorf("fail to fetch groups: %w", err)
	}

	fetched := map[LogstreamID]struct{}{}

	for _, group := range *groups.LogGroups {
		req := &model.ListLogStreamRequest{LogGroupId: group.LogGroupId}
		streams, err := m.client.ListLogStream(req)
		if err != nil {
			return fmt.Errorf("fail to fetch streams: %w", err)
		}

		for _, stream := range *streams.LogStreams {
			if _, ok := stream.Tag[streamExclusionTag]; ok {
				log.Printf("[WARN] stream '%s' of group '%s' is excluded.\n", stream.LogStreamName, group.LogGroupName)
				continue
			}

			s := NewLogstream(m.client, group, stream, stdout)
			fetched[s.ID()] = struct{}{}
			if v, ok := m.streams[s.ID()]; !ok {
				m.streams[s.ID()] = s
				go m.Queue(ctx, s)

			} else {
				v.UpdateTags(group, stream)
			}
		}
	}

	for k, s := range m.streams {
		if _, ok := fetched[k]; ok {
			continue
		}
		s.terminated = true
		delete(m.streams, k)
	}
	return
}

func (m LogstreamManager) Queue(ctx context.Context, s *Logstream) {
	d := s.start.Sub(time.Now().
		Add(-m.maxEndFromNow).
		Add(-m.minFetchRange))

	select {
	case <-time.After(d):
	case <-ctx.Done():
		return
	}

	select {
	case m.queue <- s:
	case <-ctx.Done():
		return
	}
}
