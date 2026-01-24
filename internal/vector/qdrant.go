package vector

import (
	"context"
	"log/slog"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type QdrantService struct {
	client *qdrant.Client
	col    string // 集合名称，例如 "notes_collection"
}

func NewQdrantService(host string, port int, collectionName string, apiKey string) *QdrantService {
	// 1. 基础配置
	config := &qdrant.Config{
		Host: host,
		Port: port,
	}

	// 2. 如果有密码
	if apiKey != "" {
		config.APIKey = apiKey
		config.UseTLS = false // 根据实际情况，通常本地和内网部署设为 false
	}

	// 3. 配置连接选项 [修正点]
	// 注意：这里使用的是 grpc.DialOption，而不是 qdrant.ClientOption
	if !config.UseTLS {
		config.GrpcOptions = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
	}

	// 4. 创建客户端
	client, err := qdrant.NewClient(config)
	if err != nil {
		panic("无法连接 Qdrant 数据库: " + err.Error())
	}

	svc := &QdrantService{client: client, col: collectionName}
	svc.ensureCollection()
	return svc
}

// ensureCollection 就像 LanMei 代码里那样，不存在就创建
func (s *QdrantService) ensureCollection() {
	ctx := context.Background()
	exists, err := s.client.CollectionExists(ctx, s.col)
	if err != nil {
		slog.Error("Check qdrant collection failed", "err", err)
		return
	}
	if !exists {
		err := s.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: s.col,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     1024,
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			slog.Error("Create collection failed", "err", err)
		}
	}
}

// Upsert 将向量存入 Qdrant
// id: MySQL 中的 Note ID
// vector: AI 生成的向量
func (s *QdrantService) Upsert(ctx context.Context, id uint, vector []float32, userID uint, isPrivate bool) error {
	payload := map[string]*qdrant.Value{
		"user_id":    {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(userID)}},
		"is_private": {Kind: &qdrant.Value_BoolValue{BoolValue: isPrivate}}, // 新增这个！
	}

	points := []*qdrant.PointStruct{
		{
			Id:      qdrant.NewIDNum(uint64(id)),
			Vectors: qdrant.NewVectors(vector...),
			Payload: payload, // 把 Payload 存进去
		},
	}

	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.col,
		Points:         points,
	})
	return err
}

// internal/vector/qdrant.go

func (s *QdrantService) Search(ctx context.Context, vector []float32, limit uint64, userID uint) ([]uint, error) {

	// 构造 Filter: (user_id == current_user) OR (is_public == true)
	filter := &qdrant.Filter{
		Should: []*qdrant.Condition{
			// 条件 1: 是我自己的笔记
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "user_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Integer{
								Integer: int64(userID),
							},
						},
					},
				},
			},
			// 条件 2: 是公开的笔记
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "is_private",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Boolean{
								Boolean: false,
							},
						},
					},
				},
			},
		},
	}

	res, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.col,
		Query:          qdrant.NewQuery(vector...),
		Filter:         filter, // 传入这个混合过滤器
		Limit:          &limit,
	})
	if err != nil {
		return nil, err
	}

	var ids []uint
	for _, point := range res {
		if point.Id == nil {
			continue
		}
		// 提取 ID
		if numID, ok := point.Id.PointIdOptions.(*qdrant.PointId_Num); ok {
			ids = append(ids, uint(numID.Num))
		}
	}
	return ids, nil
}
