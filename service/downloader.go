package service

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const ChunkSize int64 = 5 * 1024 * 1024 // 5MB

func ChunkDownload(url, dest string) error {
	// HEAD → 파일 크기 확인
	resp, err := http.Head(url)
	if err != nil {
		return err
	}
	resp.Body.Close()

	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil || size <= 0 {
		return fmt.Errorf("invalid content length")
	}

	partCount := int((size + ChunkSize - 1) / ChunkSize)

	client := &http.Client{}

	// chunk 다운로드
	for i := 0; i < partCount; i++ {
		partPath := fmt.Sprintf("%s.part%d", dest, i)
		if FileExists(partPath) {
			continue // 이미 받은 chunk
		}

		start := int64(i) * ChunkSize
		end := start + ChunkSize - 1
		if end >= size {
			end = size - 1
		}

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		out, err := os.Create(partPath)
		if err != nil {
			resp.Body.Close()
			return err
		}

		_, err = io.Copy(out, resp.Body)
		resp.Body.Close()
		out.Close()

		if err != nil {
			return err
		}
	}

	// 병합
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	for i := 0; i < partCount; i++ {
		partPath := fmt.Sprintf("%s.part%d", dest, i)
		in, err := os.Open(partPath)
		if err != nil {
			return err
		}
		io.Copy(out, in)
		in.Close()
		os.Remove(partPath)
	}

	return nil
}

func DownloadFile(url, dest string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
