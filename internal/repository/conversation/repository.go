package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/runtime/embedding"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/utils"
	"gorm.io/gorm"
)

type GormConversationrepo struct {
	db       *gorm.DB
	rc       *redis.Client
	msgTTL   time.Duration
	embedder embedding.Embedder // Use the embedder interface
}

func UserMsgListKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s:msgs", userID.String())
}

// CreateMemory implements types.ConversationRepository.
func (g *GormConversationrepo) CreateMemory(ctx context.Context, conversationID uuid.UUID, m types.Memory) (*types.Memory, error) {
	// Create the memory entity (without embeddings)
	me := &MemoryEntity{}
	me.FromDomain(m)

	// Start a transaction
	tx := g.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Save the main memory entity
	if err := tx.Create(&me).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Chunk the content
	chunks := g.embedder.Chunk(m.Content)
	if len(chunks) == 0 {
		tx.Rollback()
		return nil, fmt.Errorf("no chunks generated from content")
	}

	// Create embeddings for all chunks
	embeddings, err := g.embedder.Embed(ctx, chunks)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create embeddings: %v", err)
	}

	if len(embeddings) != len(chunks) {
		tx.Rollback()
		return nil, fmt.Errorf("mismatch between chunks and embeddings")
	}

	// Create and save chunk entities
	for i, chunk := range chunks {
		chunkEntity := &MemoryChunkEntity{}
		chunkEntity.FromChunk(me.ID, i, chunk, embeddings[i])

		if err := tx.Create(&chunkEntity).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to save chunk %d: %v", i, err)
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return me.ToDomain(), nil
}

// CreateMessage implements types.ConversationRepository.
func (g *GormConversationrepo) CreateMessage(ctx context.Context, userId uuid.UUID, msg types.Message) (*types.Message, error) {
	lmsg := MessageEntity{}
	lmsg.FromDomain(&msg)
	lmsg.UserID = userId // for ai responses

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
	}).Err(); err != nil {
		return nil, utils.XError{Reason: "unable to add to user msg list", Meta: err}.ToError()
	}

	return &msg, nil
}

// FetchMessage implements types.ConversationRepository.
func (g *GormConversationrepo) FetchMessage(ctx context.Context, msgId uuid.UUID) (*types.Message, error) {
	var msg MessageEntity
	// Create the proper Redis key format
	msgKey := fmt.Sprintf("msg:%s", msgId.String())
	rawMsg, err := g.rc.Get(msgKey).Result()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(rawMsg), &msg); err != nil {
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
	for _, rawKey := range rawIds {
		// rawKey is the message key like "msg:uuid", use it directly to fetch from Redis
		var msg MessageEntity
		rawMsg, err := g.rc.Get(rawKey).Result()
		if err != nil {
			continue // message might have expired
		}

		if err := json.Unmarshal([]byte(rawMsg), &msg); err != nil {
			continue // corrupted data
		}

		msgs = append(msgs, *msg.ToDomain())
	}
	return msgs, nil
}

// FindMemories implements types.ConversationRepository.
func (g *GormConversationrepo) FindMemories(ctx context.Context, conversationID uuid.UUID, msr types.MemorySearchRequest) ([]types.Memory, error) {
	// first check for vector search
	if msr.QueryStatement != nil {
		// Create embedding for the query
		queryVec, err := g.embedder.EmbedSingle(ctx, *msr.QueryStatement)
		if err != nil {
			return nil, err
		}

		// Search through memory chunks, but return unique memories
		sql := `
            SELECT DISTINCT m.*, VEC_COSINE_DISTANCE(c.embedding_ref, ?) AS distance
            FROM memory_entities m
            JOIN memory_chunk_entities c ON m.id = c.memory_id
            WHERE m.conversation_id = ?
            ORDER BY distance
            LIMIT 10
        `
		var entities []MemoryEntity
		if err := g.db.Raw(sql, queryVec, conversationID).Scan(&entities).Error; err != nil {
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
	if err := g.db.WithContext(ctx).Where("owner_id = ?", userID).First(&conv).Error; err != nil {
		// create new conv then
		conv = ConversationEntity{
			ID:        uuid.New(),
			OwnerID:   userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := g.db.WithContext(ctx).Create(&conv).Error; err != nil {
			return nil, err
		}
	}
	// fetch all msgs
	if csr.MsgSearch == nil {
		start := int64(0)
		end := int64(-1) // -1 means fetch until the end (all messages)
		csr.MsgSearch = &utils.Range[int64]{Min: &start, Max: &end}
	}
	msgs, _ := g.FetchUserMessages(ctx, userID, int64(*csr.MsgSearch.Min), int64(*csr.MsgSearch.Max))
	// fetch memories separately
	mems, _ := g.FindMemories(ctx, conv.ID, *csr.Msr)
	dconv := conv.ToDomain()
	dconv.Messages = append(dconv.Messages, msgs...)
	dconv.Memories = append(dconv.Memories, mems...)
	return &dconv, nil
}

func NewGormConvoRepo(db *gorm.DB, rc *redis.Client, msgTTL time.Duration, embedder embedding.Embedder) types.ConversationRepository {
	return &GormConversationrepo{
		db: db, rc: rc, msgTTL: msgTTL, embedder: embedder,
	}
}
