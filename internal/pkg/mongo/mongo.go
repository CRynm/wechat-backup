package mongo

import (
	"context"
	"github.com/marmotedu/log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	URI         string
	Database    string
	Timeout     time.Duration
	MaxPoolSize uint64
	MinPoolSize uint64
	MaxIdleTime time.Duration
	RetryWrites bool
	RetryReads  bool
}

type DB struct {
	client   *mongo.Client
	database *mongo.Database
}

var (
	instance *DB
	once     sync.Once
)

// GetMongoDB 获取MongoDB实例
func GetMongoDB() *DB {
	if instance == nil {
		log.Fatal("MongoDB未初始化")
	}
	return instance
}

// InitMongoDB 初始化MongoDB连接
func InitMongoDB(config Config) error {
	var err error

	once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
		defer cancel()

		// 配置连接选项
		opts := options.Client().
			ApplyURI(config.URI).
			SetMaxPoolSize(config.MaxPoolSize).
			SetMinPoolSize(config.MinPoolSize).
			SetMaxConnIdleTime(config.MaxIdleTime).
			SetRetryWrites(config.RetryWrites).
			SetRetryReads(config.RetryReads)

		// 连接MongoDB
		client, err := mongo.Connect(ctx, opts)
		if err != nil {
			return
		}

		// 测试连接
		if err = client.Ping(ctx, nil); err != nil {
			return
		}

		instance = &DB{
			client:   client,
			database: client.Database(config.Database),
		}
	})

	return err
}

// Collection 获取集合
func (m *DB) Collection(name string) *mongo.Collection {
	return m.database.Collection(name)
}

// Close 关闭连接
func (m *DB) Close() {
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.client.Disconnect(ctx); err != nil {
			log.Errorf("MongoDB关闭连接失败: %+v", err)
		}
	}
}
