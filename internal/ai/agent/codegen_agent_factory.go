package agent

import (
	"context"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"strconv"
	"sync"
	"time"
	"yikou-ai-go-teach/internal/ai/llm"
	"yikou-ai-go-teach/internal/service"
	"yikou-ai-go-teach/internal/store"
	"yikou-ai-go-teach/pkg/enum"
)

const MaxAgentInstances = 1000

var (
	serviceCache    = cache.New(30*time.Minute, 10*time.Minute)
	instanceCount   int
	instanceCountMu sync.Mutex
)

type CodeGenAgentFactory struct {
	chatModel          *llm.ChatModelWrapper
	redisClient        *redis.Client
	chatHistoryService service.IChatHistoryService
}

func NewCodeGenAgentFactory(chatModel *llm.ChatModelWrapper,
	redisClient *redis.Client, chatHistoryService service.IChatHistoryService) *CodeGenAgentFactory {
	serviceCache.OnEvicted(func(k string, v interface{}) {
		logger.Debugf("AI服务实例被移除，缓冲键: %v", k)
	})
	return &CodeGenAgentFactory{
		chatModel:          chatModel,
		redisClient:        redisClient,
		chatHistoryService: chatHistoryService,
	}
}

func (c CodeGenAgentFactory) evictOldest() {
	items := serviceCache.Items()
	oldestKey := ""
	var oldestExpiration int64

	for k, item := range items {
		if item.Expiration == 0 {
			continue
		}
		if oldestKey == "" || item.Expiration < oldestExpiration {
			oldestExpiration = item.Expiration
			oldestKey = k
		}
	}
	if oldestKey != "" {
		serviceCache.Delete(oldestKey)
		instanceCountMu.Lock()
		instanceCount--
		instanceCountMu.Unlock()
	}
}

func buildCacheKey(appId int64, codeGenType enum.CodeGenTypeEnum) string {
	return strconv.Itoa(int(appId)) + "_" + string(codeGenType)
}

func (c CodeGenAgentFactory) GetCodeGenAgent(appId int64, codeGenType enum.CodeGenTypeEnum) (*CodeGenAgent, error) {
	redisStore := store.NewRedisMemoryStore(c.redisClient, strconv.Itoa(int(appId)), 20, 24*time.Hour)
	_, err := c.chatHistoryService.LoadChatHistoryToMemory(context.Background(), appId, redisStore, 20)
	if err != nil {
		return nil, err
	}
	key := buildCacheKey(appId, codeGenType)
	if agent, found := serviceCache.Get(key); found {
		return agent.(*CodeGenAgent), nil
	}
	instanceCountMu.Lock()
	if instanceCount >= MaxAgentInstances {
		c.evictOldest()
	}
	instanceCountMu.Unlock()
	agent := NewCodeGenAgent(c.chatModel, codeGenType, redisStore)

	serviceCache.Set(key, agent, cache.DefaultExpiration)
	instanceCountMu.Lock()
	instanceCount++
	instanceCountMu.Unlock()
	return agent, nil
}
