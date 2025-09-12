package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/database/dbtypes"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/utils"
	"gorm.io/gorm"
)

type GormConversationrepo struct {
	db        *gorm.DB
	rc        *redis.Client
	msgTTL    time.Duration
	embedFunc func(ctx context.Context, text string) ([]float32, error) // from a provider
}

func UserMsgListKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s:msgs", userID.String())
}

// CreateMemory implements types.ConversationRepository.
func (g *GormConversationrepo) CreateMemory(ctx context.Context, conversationID uuid.UUID, m types.Memory) (*types.Memory, error) {
	me := &MemoryEntity{}
	embed, err := g.embedFunc(ctx, m.Content)
	if err != nil {
		return nil, err
	}
	me.FromDomain(m, embed)
	// store in db
	if err := g.db.WithContext(ctx).Create(&me).Error; err != nil {
		return nil, err
	}
	return me.ToDomain(), nil
}

// CreateMessage implements types.ConversationRepository.
func (g *GormConversationrepo) CreateMessage(ctx context.Context, userId uuid.UUID, msg types.Message) (*types.Message, error) {
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

// FetchMessage implements types.ConversationRepository.
func (g *GormConversationrepo) FetchMessage(ctx context.Context, msgId uuid.UUID) (*types.Message, error) {
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

// FetchUserMessages implements types.ConversationRepository.
func (g *GormConversationrepo) FetchUserMessages(ctx context.Context, userId uuid.UUID, start, end int64) ([]types.Message, error) {
	// fetch from redis store
	var msgs []types.Message
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

// FindMemories implements types.ConversationRepository.
func (g *GormConversationrepo) FindMemories(ctx context.Context, conversationID uuid.UUID, msr types.MemorySearchRequest) ([]types.Memory, error) {
	// first check for vector search
	if msr.QueryStatement != nil {
		queryVec, err := g.embedFunc(ctx, *msr.QueryStatement)
		if err != nil {
			return nil, err
		}
		sql := `
            SELECT *, VEC_COSINE_DISTANCE(embedding_ref, ?) AS distance
            FROM memory_entities
            WHERE conversation_id = ?
            ORDER BY distance
            LIMIT 10
        `
		var entities []MemoryEntity
		if err := g.db.Raw(sql, dbtypes.XVector(queryVec), conversationID).Scan(&entities).Error; err != nil {
			return nil, err
		}
		var dms []types.Memory
		for _, m := range entities {
			dms = append(dms, *m.ToDomain())
		}
		return dms, nil
	}

	base := g.db.WithContext(ctx).Model(&MemoryEntity{}).Where("conversation_id = ?", conversationID)
	// Optional saliency filter
	if msr.SaliencyRange != nil {
		base = base.Where("saliency_score BETWEEN ? AND ?", msr.SaliencyRange.Min, msr.SaliencyRange.Max)
	}

	// Optional time filter
	if msr.WithinPeriod != nil {
		base = base.Where("created_at BETWEEN ? AND ?", msr.WithinPeriod.Min, msr.WithinPeriod.Max)
	}

	var entities []MemoryEntity
	if err := base.Find(&entities).Error; err != nil {
		return nil, err
	}
	var dms []types.Memory
	for _, m := range entities {
		dms = append(dms, *m.ToDomain())
	}
	return dms, nil

}

// RetrieveUserConversation implements types.ConversationRepository.
func (g *GormConversationrepo) RetrieveUserConversation(ctx context.Context, userID uuid.UUID, csr *types.ConvFetchRequest) (*types.Conversation, error) {
	// fetch conversation
	var conv ConversationEntity
	if err := g.db.WithContext(ctx).Where("OwnerID = ?", userID).Preload("Memories").First(&conv).Error; err != nil {
		// create new conv then
		conv = ConversationEntity{
			ID:        uuid.New(),
			OwnerID:   userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := g.db.WithContext(ctx).Create(conv).Error; err != nil {
			return nil, err
		}
	}
	// fetch all msgs
	msgs, _ := g.FetchUserMessages(ctx, userID, int64(*csr.MsgSearch.Min), int64(*csr.MsgSearch.Max))
	// mems, _ := g.FindMemories(ctx, conv.ID, *csr.Msr)
	dconv := conv.ToDomain()
	dconv.Messages = append(dconv.Messages, msgs...)
	return &dconv, nil
}

func NewGormConvoRepo(db *gorm.DB, rc *redis.Client, msgTTL time.Duration) types.ConversationRepository {
	return &GormConversationrepo{
		db: db, rc: rc, msgTTL: msgTTL,
	}
}
