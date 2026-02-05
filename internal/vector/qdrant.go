package vector

import (
	"context"

	"github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type QdrantService struct {
	client *qdrant.Client
	col    string
}

func NewQdrantService(host string, port int, collectionName string, apiKey string) *QdrantService {
	config := &qdrant.Config{
		Host: host,
		Port: port,
	}

	if apiKey != "" {
		config.APIKey = apiKey
		config.UseTLS = false // 根据实际情况，通常本地和内网部署设为 false
	}

	if !config.UseTLS {
		config.GrpcOptions = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
	}

	client, err := qdrant.NewClient(config)
	if err != nil {
		panic("无法连接 Qdrant 数据库: " + err.Error())
	}

	svc := &QdrantService{client: client, col: collectionName}
	svc.ensureCollection()
	return svc
}

// ensureCollection 不存在就创建
func (s *QdrantService) ensureCollection() {
	ctx := context.Background()
	exists, err := s.client.CollectionExists(ctx, s.col)
	if err != nil {
		zap.L().Error("Check qdrant collection failed", zap.Error(err))
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
			zap.L().Error("Create collection failed", zap.Error(err))
		}
	}
}

// Upsert 将向量存入 Qdrant
// id: MySQL 中的 Note ID
// vector: AI 生成的向量
func (s *QdrantService) Upsert(ctx context.Context, id uint, vector []float32, userID uint, isPrivate bool) error {
	payload := map[string]*qdrant.Value{
		"user_id":    {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(userID)}},
		"is_private": {Kind: &qdrant.Value_BoolValue{BoolValue: isPrivate}},
	}

	points := []*qdrant.PointStruct{
		{
			Id:      qdrant.NewIDNum(uint64(id)),
			Vectors: qdrant.NewVectors(vector...),
			Payload: payload,
		},
	}

	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.col,
		Points:         points,
	})
	return err
}

func (s *QdrantService) Search(ctx context.Context, vector []float32, limit uint64, userID uint) ([]uint, error) {
	// 构造 Filter: (user_id == current_user) OR (is_public == true)
	filter := &qdrant.Filter{
		Should: []*qdrant.Condition{
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
