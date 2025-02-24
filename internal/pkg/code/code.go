package code

import (
    "github.com/marmotedu/errors"
)

var (
    ErrProxyServerStart = errors.New("Failed to start proxy server")
    ErrDatabase        = errors.New("Database operation failed")
    ErrInvalidConfig   = errors.New("Invalid configuration")
    ErrBackupFailed    = errors.New("Backup operation failed")
)
