// Copyright 2023 Linkall Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/vanus-labs/connector/source/chatai/chat/ernie_bot"
	"github.com/vanus-labs/connector/source/chatai/chat/gpt"
	"github.com/vanus-labs/connector/source/chatai/chat/model"
)

const (
	responseEmpty = "Get response empty."
	responseErr   = "Get response failed."
)

var (
	ErrLimit = fmt.Errorf("reached the daily limit")
)

type ChatClient interface {
	SendChatCompletion(ctx context.Context, userIdentifier, content string) (string, error)
	SendChatCompletionStream(ctx context.Context, userIdentifier, content string) (model.ChatCompletionStream, error)
	Reset()
}

type Type string

const (
	ChatGPT      Type = "chatgpt"
	ChatErnieBot Type = "wenxin"
)

type ChatService struct {
	chatGpt      ChatClient
	ernieBot     ChatClient
	config       ChatConfig
	lock         sync.RWMutex
	day          int
	limitContent string
	userNum      map[string]int
	ctx          context.Context
	cancel       context.CancelFunc
	logger       zerolog.Logger
}

func NewChatService(config ChatConfig, logger zerolog.Logger) *ChatService {
	config.init()
	s := &ChatService{
		logger:       logger,
		config:       config,
		userNum:      map[string]int{},
		chatGpt:      gpt.NewChatGPTService(config.GPT, config.MaxTokens, config.EnableContext),
		ernieBot:     ernie_bot.NewErnieBotService(config.ErnieBot, config.MaxTokens, config.EnableContext),
		day:          today(),
		limitContent: fmt.Sprintf("You've reached the daily limit (%d/day). Your quota will be restored tomorrow.", config.EverydayLimit),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	go func() {
		now := time.Now().UTC()
		next := now.Add(time.Hour)
		next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), 0, 0, 0, next.Location())
		t := time.NewTicker(next.Sub(now))
		select {
		case <-s.ctx.Done():
			t.Stop()
			return
		case <-t.C:
			s.reset()
		}
		t.Stop()
		tk := time.NewTicker(time.Hour)
		defer tk.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-tk.C:
				s.reset()
			}
		}
	}()
	return s
}

func today() int {
	return time.Now().UTC().Day()
}

func (s *ChatService) Close() {
	s.cancel()
}

func (s *ChatService) addNum(userIdentifier string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	num, ok := s.userNum[userIdentifier]
	if !ok {
		num = 0
	}
	num++
	s.userNum[userIdentifier] = num
}

func (s *ChatService) getNum(userIdentifier string) int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	num, ok := s.userNum[userIdentifier]
	if !ok {
		return 0
	}
	return num
}

func (s *ChatService) reset() {
	s.lock.Lock()
	defer s.lock.Unlock()
	time.Sleep(time.Second)
	t := today()
	if s.day == t {
		return
	}
	s.day = t
	s.userNum = map[string]int{}
	s.chatGpt.Reset()
	s.ernieBot.Reset()
}

func (s *ChatService) ChatCompletion(ctx context.Context, chatType Type, userIdentifier, content string) (resp string, err error) {
	if chatType == "" {
		chatType = s.config.DefaultChatMode
	}
	num := s.getNum(userIdentifier)
	if num >= s.config.EverydayLimit {
		return s.limitContent, ErrLimit
	}
	s.logger.Info().Interface("chat", chatType).Str("user", userIdentifier).Msg("receive content:" + content)
	switch chatType {
	case ChatErnieBot:
		resp, err = s.ernieBot.SendChatCompletion(ctx, userIdentifier, content)
	case ChatGPT:
		resp, err = s.chatGpt.SendChatCompletion(ctx, userIdentifier, content)
	}
	if err != nil {
		return responseErr, err
	}
	if resp == "" {
		return responseEmpty, nil
	}
	s.addNum(userIdentifier)
	return resp, nil
}

func (s *ChatService) ChatCompletionStream(ctx context.Context, chatType Type, userIdentifier, content string) (stream model.ChatCompletionStream, err error) {
	if chatType == "" {
		chatType = s.config.DefaultChatMode
	}
	num := s.getNum(userIdentifier)
	if num >= s.config.EverydayLimit {
		return nil, fmt.Errorf("you've reached the daily limit (%d/day). Your quota will be restored tomorrow", s.config.EverydayLimit)
	}
	s.logger.Info().Str("user", userIdentifier).Msg("receive content:" + content)
	switch chatType {
	case ChatErnieBot:
		stream, err = s.ernieBot.SendChatCompletionStream(ctx, userIdentifier, content)
	case ChatGPT:
		stream, err = s.chatGpt.SendChatCompletionStream(ctx, userIdentifier, content)
	}
	if err != nil {
		return nil, err
	}
	s.addNum(userIdentifier)
	return stream, nil
}
