// Package project provides the core functionality for managing projects within the gocli application.
package project

import (
	log2 "github.com/yeisme/gocli/pkg/utils/log"
)

var log log2.Logger

func init() {
	log = log2.GetLogger()
}
