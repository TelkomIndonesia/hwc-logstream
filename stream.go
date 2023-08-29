package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/lts/v2/model"
)

type LogstreamClient interface {
	ListLogs(*model.ListLogsRequest) (*model.ListLogsResponse, error)
	CreateTags(*model.CreateTagsRequest) (*model.CreateTagsResponse, error)
}

type LogPrinter interface {
	Println(v ...any)
}

type LogstreamID struct {
	groupID  string
	streamID string
}

type Logstream struct {
	id     LogstreamID
	client LogstreamManagerClient
	group  model.LogGroup
	stream model.LogStream
	tags   map[string]string

	limit      int32
	start      time.Time
	last       time.Time
	terminated bool

	out LogPrinter
}

func NewLogstream(c LogstreamManagerClient, group model.LogGroup, stream model.LogStream, out *log.Logger) (s *Logstream) {
	s = &Logstream{
		client: c,
		group:  group,
		stream: stream,
		id: LogstreamID{
			groupID:  group.LogGroupId,
			streamID: stream.LogStreamId,
		},
		limit: 100,
		out:   out,
	}
	s.UpdateTags(group, stream)
	if err := s.LoadPositition(); err != nil {
		log.Printf("[WARN] failed to load stored position: %v\n", err)
	}
	return s
}

func (s Logstream) ID() LogstreamID {
	return s.id
}

func (s *Logstream) String() string {
	return fmt.Sprintf("%s:%s", s.group.LogGroupName, s.stream.LogStreamName)
}

func (s *Logstream) UpdateTags(group model.LogGroup, stream model.LogStream) {
	tags := map[string]string{}
	for k, v := range group.Tag {
		if strings.HasPrefix(k, "_") || k == streamExclusionTag {
			continue
		}
		tags[k] = v
	}
	for k, v := range stream.Tag {
		if strings.HasPrefix(k, "_") || k == streamExclusionTag {
			continue
		}
		tags[k] = v
	}
	if s.tags != nil {
		tags[streamPosTag] = s.tags[streamPosTag]
	}
	s.tags = tags
}

func (s *Logstream) LoadPositition() (err error) {
	s.start = time.Now()
	s.last = s.start

	v, ok := s.tags[streamPosTag]
	if !ok {
		return
	}

	t, err := parseTime(&v)
	if err != nil {
		return fmt.Errorf("fail to parse %s as time: %w", v, err)
	}

	delete(s.tags, streamPosTag)
	s.start, s.last = t, t
	return
}

func (s *Logstream) Stream(ctx context.Context, end time.Time) (err error) {
	for end.Sub(s.start) > time.Second && !s.terminated && ctx.Err() == nil {
		l, err := s.FetchNext(end)
		if err != nil {
			return fmt.Errorf("fail to stream: %w", err)
		}

		var logs []model.LogContents
		if l != nil && *l.Logs != nil {
			logs = *l.Logs
		}
		for _, l := range logs {
			s.Print(l)
		}
		if err := s.SavePositition(); err != nil {
			log.Printf("[WARN] fail to save position of %s using tags: %v\n", s, err)
		}
	}

	return
}

func (s *Logstream) FetchNext(end time.Time) (logs *model.ListLogsResponse, err error) {
	ln := strconv.FormatInt(s.start.UnixNano(), 10)
	logs, err = s.client.ListLogs(&model.ListLogsRequest{
		LogGroupId:  s.group.LogGroupId,
		LogStreamId: s.stream.LogStreamId,
		Body: &model.QueryLtsLogParams{
			StartTime: strconv.FormatInt(s.start.UnixNano()/1000000, 10),
			EndTime:   strconv.FormatInt(end.UnixNano()/1000000, 10),
			LineNum:   &ln,
			Limit:     &s.limit,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("fail to fetch: %w", err)
	}

	s.last = end
	if logs.Logs == nil || len(*logs.Logs) == 0 {
		return nil, nil
	}

	lnum := (*logs.Logs)[len(*logs.Logs)-1].LineNum
	if lnum == nil {
		return
	}

	t, err := parseTime(lnum)
	if err != nil {
		return nil, fmt.Errorf("fail to parse %s as time: %w", *lnum, err)
	}
	s.last = t
	return
}

func (s *Logstream) Print(l model.LogContents) {
	t, _ := parseTime(l.LineNum)
	d := map[string]interface{}{
		"message":   l.Content,
		"labels":    l.Labels,
		"tags":      s.tags,
		"timestamp": t,
	}
	b, err := json.Marshal(d)
	if err != nil {
		log.Printf("[WARN] encode log from %s failed: %v\n", s, err)
	}
	s.out.Println(string(b))
}

func parseTime(lineNum *string) (t time.Time, err error) {
	if lineNum == nil {
		return
	}

	i, err := strconv.ParseInt(*lineNum, 10, 64)
	if err != nil {
		return
	}
	return time.Unix(0, i), nil
}

func (s *Logstream) SavePositition() (err error) {
	tags := map[string]string{}
	for k, v := range s.stream.Tag {
		tags[k] = v
	}
	pos := strconv.FormatInt(s.last.UnixNano(), 10)
	tags[streamPosTag] = pos

	tagsb := []model.TagsBody{}
	for k, v := range tags {
		tagsb = append(tagsb, model.TagsBody{Key: &k, Value: &v})
	}
	_, err = s.client.CreateTags(&model.CreateTagsRequest{
		ResourceType: "topics",
		ResourceId:   s.stream.LogStreamId,
		Body: &model.CreateTagsReqbody{
			Tags:   tagsb,
			Action: "create",
			IsOpen: false,
		},
	})
	if err != nil {
		return
	}

	s.start = s.last
	return
}
