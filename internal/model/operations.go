package model

import (
    "context"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo/options"
)

// CreateProfile 创建新的用户档案
func CreateProfile(profile *Profile) error {
    profile.CreatedAt = time.Now()
    profile.UpdatedAt = time.Now()
    
    collection := GetCollection("profiles")
    _, err := collection.InsertOne(context.Background(), profile)
    return err
}

// FindProfileByWechatID 通过微信ID查找用户档案
func FindProfileByWechatID(wechatID string) (*Profile, error) {
    collection := GetCollection("profiles")
    
    var profile Profile
    err := collection.FindOne(context.Background(), bson.M{"wechat_id": wechatID}).Decode(&profile)
    if err != nil {
        return nil, err
    }
    return &profile, nil
}

// CreateMessage 创建新消息
func CreateMessage(message *Message) error {
    message.CreatedAt = time.Now()
    message.UpdatedAt = time.Now()
    
    collection := GetCollection("messages")
    _, err := collection.InsertOne(context.Background(), message)
    return err
}

// CreateMedia 创建新的媒体记录
func CreateMedia(media *Media) error {
    media.CreatedAt = time.Now()
    media.UpdatedAt = time.Now()
    
    collection := GetCollection("media")
    _, err := collection.InsertOne(context.Background(), media)
    return err
}

// FindMessagesByProfileID 查找用户的所有消息
func FindMessagesByProfileID(profileID primitive.ObjectID) ([]*Message, error) {
    collection := GetCollection("messages")
    
    opts := options.Find().SetSort(bson.D{{Key: "send_time", Value: 1}})
    cursor, err := collection.Find(context.Background(), bson.M{"profile_id": profileID}, opts)
    if err != nil {
        return nil, err
    }
    
    var messages []*Message
    err = cursor.All(context.Background(), &messages)
    return messages, err
} 