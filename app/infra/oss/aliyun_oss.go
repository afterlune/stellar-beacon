package oss

import (
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/shared"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"io"
	"log/slog"
	"mime/multipart"
	"strings"
)

func Upload(file *multipart.FileHeader, path string) string {
	c := getOssClient()
	open, err := file.Open()
	if err != nil {
		slog.Error("open uploaded file failed", "error", err)
	}

	index := strings.LastIndex(file.Filename, ".")
	preSuffix := file.Filename[:index]
	postSuffix := file.Filename[index:]
	fileName := shared.GetMD5(preSuffix)

	exist, err := c.IsObjectExist(path + fileName + postSuffix)
	if err != nil {
		slog.Error("check OSS object failed", "error", err)
	}
	if !exist {
		err = c.PutObject(path+fileName+postSuffix, io.Reader(open))
		if err != nil {
			slog.Error("upload OSS object failed", "error", err)
		}
	}
	return path + fileName + postSuffix
}

func getOssClient() *oss.Bucket {
	cfg := new(config.Oss).Oss()
	client, err := oss.New(cfg.EndPoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		slog.Error("create OSS client failed", "error", err)
	}

	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		slog.Error("open OSS bucket failed", "error", err)
	}

	return bucket
}

func UploadFile(value io.Reader, fileName, path string) string {
	c := getOssClient()
	exist, err := c.IsObjectExist(path + fileName)
	if err != nil {
		slog.Error("check OSS object failed", "error", err)
	}
	if !exist {
		err = c.PutObject(path+fileName, value)
		if err != nil {
			slog.Error("upload OSS object failed", "error", err)
		}
	}
	return path + fileName
}
