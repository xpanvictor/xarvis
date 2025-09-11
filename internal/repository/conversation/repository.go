package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/pkg/utils"
	"gorm.io/gorm"
)

type GormConversationrepo struct {
	db     *gorm.DB
	rc     *redis.Client
	msgTTL time.Duration
}

func UserMsgListKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s:msgs", userID.String())
}

// CreateMemory implements conversation.ConversationRepository.
func (g *GormConversationrepo) CreateMemory(ctx context.Context, conversationID uuid.UUID, m conversation.Memory) (*conversation.Memory, error) {
	panic("unimplemented")
}

// CreateMessage implements conversation.ConversationRepository.
func (g *GormConversationrepo) CreateMessage(ctx context.Context, userId uuid.UUID, msg conversation.Message) (*conversation.Message, error) {
	lmsg := MessageEntity{}
	lmsg.FromDomain(&msg)

	data, err := json.Marshal(lmsg)
	if err != nil {
		return nil, fmt.Errorf("can't marshal msg")
	}

	mk := lmsg.Key()
	if err := g.rc.Set(mk, data, g.msgTTL).Err(); err != nil {
		return nil, utils.XError{Reason: "storing user msg", Meta: err}.ToError()
	}

	score := float64(lmsg.Timestamp.Unix())
	if err := g.rc.ZAdd(UserMsgListKey(msg.UserId), redis.Z{
		Member: lmsg.Key(),
		Score:  score,
	}); err != nil {
		return nil, utils.XError{Reason: "unable to add to user msg list", Meta: err}.ToError()
	}

	return &msg, nil
}

// FetchMessage implements conversation.ConversationRepository.
func (g *GormConversationrepo) FetchMessage(ctx context.Context, msgId uuid.UUID) (*conversation.Message, error) {
	var msg MessageEntity
	rawMsg, err := g.rc.Get(msgId.String()).Result()
	if err != nil {
		return nil, err
	}

	if err != json.Unmarshal([]byte(rawMsg), &msg) {
		return nil, err
	}

	return msg.ToDomain(), nil
}

// FetchUserMessages implements conversation.ConversationRepository.
func (g *GormConversationrepo) FetchUserMessages(ctx context.Context, userId uuid.UUID, start, end int64) ([]conversation.Message, error) {
	// fetch from redis store
	var msgs []conversation.Message
	rawIds, err := g.rc.ZRange(UserMsgListKey(userId), start, end).Result()
	if err != nil {
		return nil, err
	}
	for _, rawId := range rawIds {
		if id, err := uuid.Parse(rawId); err != nil {
			msg, err := g.FetchMessage(ctx, id)
			if err != nil {
				continue
			}
			msgs = append(msgs, *msg)
		} else {
			continue
		}
	}
	return msgs, nil
}

// FindMemories implements conversation.ConversationRepository.
func (g *GormConversationrepo) FindMemories(conversationID uuid.UUID, msr conversation.MemorySearchRequest) ([]conversation.Memory, error) {
	panic("unimplemented")
}

// RetrieveUserConversation implements conversation.ConversationRepository.
func (g *GormConversationrepo) RetrieveUserConversation(userID uuid.UUID) (*conversation.Conversation, error) {
	panic("unimplemented")
}

func NewGormConvoRepo(db *gorm.DB, rc *redis.Client, msgTTL time.Duration) conversation.ConversationRepository {
	return &GormConversationrepo{db: db, rc: rc, msgTTL: msgTTL}
}
