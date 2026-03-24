package splithttp

import (
	"io"
	"net/http"
	"sync"
	"time"
)

// SeamlessDownlinkReader скрывает разрывы GET-соединения от ядра Xray
type SeamlessDownlinkReader struct {
	client     *http.Client
	reqFactory func() (*http.Request, error) // Функция для генерации нового GET-запроса
	body       io.ReadCloser
	mu         sync.Mutex
}

func (r *SeamlessDownlinkReader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for {
		if r.body == nil {
			// Тело пустое, нужно переподключиться
			req, err := r.reqFactory()
			if err != nil {
				return 0, err
			}

			resp, err := r.client.Do(req)
			if err != nil {
				// Если сеть реально упала, ждем секунду и пробуем снова
				time.Sleep(1 * time.Second)
				continue
			}
			r.body = resp.Body
		}

		n, err = r.body.Read(p)

		if err == io.EOF {
			// СЕРВЕР ПРИНУДИТЕЛЬНО ЗАКРЫЛ GET
			r.body.Close()
			r.body = nil

			if n > 0 {
				return n, nil
			}
			continue
		}

		return n, err
	}
}

func (r *SeamlessDownlinkReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.body != nil {
		return r.body.Close()
	}
	return nil
}
