package handler

import (
	"fmt"
	"github.com/labstack/echo/v5"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// UploadImage 处理上传图片的请求 todo 多文件上传
func Upload(c *echo.Context) error {
	files, err := c.MultipartForm()
	if err != nil {
		return err
	}

	data := []map[string]string{}
	for _, file := range files.File["files"] {
		// 打开上传的文件
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		// 读取文件的字节数据
		imgByte, err := io.ReadAll(src)
		if err != nil {
			return err
		}
		// 使用 crc32 和日期重命名文件
		renameFilePath := filepath.Join("usr", "uploads",
			fmt.Sprintf("%d/%02d/%v%s",
				time.Now().Year(),
				time.Now().Month(),
				crc32.ChecksumIEEE(imgByte),
				filepath.Ext(file.Filename)),
		)
		if c.QueryParam("type") == "cover" {
			renameFilePath = filepath.Join("usr", "uploads", "Background", "Cover", file.Filename)
		}
		// 创建文件路径中的所有目录
		if err := os.MkdirAll(filepath.Dir(renameFilePath), 0755); err != nil {
			return err
		}
		// 打开重命名后的文件
		f, err := os.OpenFile(renameFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		// 将压缩后的字节写入文件
		if _, err := f.Write(imgByte); err != nil {
			return err
		}
		data = append(data, map[string]string{
			"url":   "/" + filepath.ToSlash(renameFilePath),
			"alt":   file.Filename,
			"title": file.Filename,
		})
	}

	// 返回成功响应，包括重命名后的文件路径、原始文件名作为 alt 和 title
	return c.JSON(http.StatusOK, data)
}
