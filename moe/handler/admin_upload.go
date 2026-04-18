package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
)

// Upload 处理图片/封面上传。
//   - 普通上传：按 sha256 前 16 位去重，保存到 usr/uploads/<year>/<month>/
//   - 封面上传（type=cover）：保存到 usr/uploads/background/
func Upload(c *echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		return err
	}

	isCover := c.QueryParam("type") == "cover"
	now := time.Now()
	var data []map[string]string

	for _, file := range form.File["files"] {
		// 只取 basename，避免 "../" 穿越
		safeName := filepath.Base(file.Filename)
		if safeName == "." || safeName == ".." || safeName == "" || strings.ContainsAny(safeName, `/\`) {
			return echo.NewHTTPError(http.StatusBadRequest, "非法文件名")
		}

		savePath, err := saveUpload(file.Open, safeName, isCover, now)
		if err != nil {
			return err
		}
		data = append(data, map[string]string{
			"url":   "/" + filepath.ToSlash(savePath),
			"alt":   safeName,
			"title": safeName,
		})
	}

	return c.JSON(http.StatusOK, data)
}

// saveUpload 以流式方式落盘，同时算出 sha256 作为去重文件名。
// 封面上传保留原文件名。
func saveUpload(open func() (multipart.File, error), safeName string, isCover bool, now time.Time) (string, error) {
	src, err := open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	var dst string
	if isCover {
		dst = filepath.Join("usr", "uploads", "background", safeName)
	} else {
		// 先写临时文件 + 算 sha256，再重命名到最终路径
		tmp, err := os.CreateTemp("", "smoe-upload-*")
		if err != nil {
			return "", err
		}
		tmpPath := tmp.Name()
		defer os.Remove(tmpPath)

		h := sha256.New()
		if _, err := io.Copy(io.MultiWriter(tmp, h), src); err != nil {
			tmp.Close()
			return "", err
		}
		if err := tmp.Close(); err != nil {
			return "", err
		}

		digest := hex.EncodeToString(h.Sum(nil))[:16]
		dst = filepath.Join("usr", "uploads",
			fmt.Sprintf("%d/%02d/%s%s", now.Year(), now.Month(), digest, filepath.Ext(safeName)))

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return "", err
		}
		// 同一 sha256 的文件直接复用，不重复写
		if _, err := os.Stat(dst); err == nil {
			return dst, nil
		}
		if err := os.Rename(tmpPath, dst); err != nil {
			// 跨卷时 rename 可能失败，回退到拷贝
			if err := copyFile(tmpPath, dst); err != nil {
				return "", err
			}
		}
		return dst, nil
	}

	// 封面分支：直接 truncate 写入
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, src); err != nil {
		return "", err
	}
	return dst, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

