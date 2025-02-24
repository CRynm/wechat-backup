package model

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var database *mongo.Database

// InitDB 初始化数据库连接
func InitDB(mongoURI string, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return err
	}

	// 测试连接
	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	database = client.Database(dbName)
	log.Println("Connected to MongoDB!")
	return nil
}

// GetCollection 获取集合
func GetCollection(name string) *mongo.Collection {
	return database.Collection(name)
}

// CloseDB 关闭数据库连接
func CloseDB() {
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}
}