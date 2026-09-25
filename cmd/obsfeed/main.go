// Command obsfeed reads an OBS RTMP stream, converts its audio to PCM and feeds one session.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const audioChunkBytes = 3200 // 100 ms of PCM16 mono at 16 kHz.

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Getenv("OPERATOR_TOKEN")); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string, token string) error {
	flags := flag.NewFlagSet("obs-feed", flag.ContinueOnError)
	base := flags.String("url", "http://127.0.0.1:8080", "Sintonizados gateway URL")
	sessionID := flags.String("session", "", "stable session ID; also used as the OBS stream key")
	title := flags.String("title", "Charla", "talk title")
	rtmpInput := flags.String("rtmp-url", "", "RTMP input URL; defaults to rtmp://127.0.0.1:1935/live/<session>")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *sessionID == "" {
		return errors.New("use -session ID and start OBS publishing to /live/ID before this command")
	}
	if token == "" {
		return errors.New("OPERATOR_TOKEN is required")
	}
	if *rtmpInput == "" {
		*rtmpInput = "rtmp://127.0.0.1:1935/live/" + *sessionID
	}
	parsed, err := url.Parse(*rtmpInput)
	if err != nil || (parsed.Scheme != "rtmp" && parsed.Scheme != "rtmps") || parsed.Host == "" {
		return errors.New("-rtmp-url must be an RTMP or RTMPS URL")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	*base = strings.TrimRight(*base, "/")
	createBody, _ := json.Marshal(map[string]string{"session_id": *sessionID, "title": *title, "language": "auto"})
	status, body, err := gatewayRequest(ctx, client, token, *base+"/api/sessions", "application/json", createBody, 0)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	if status != http.StatusCreated {
		return fmt.Errorf("create session: HTTP %d %s", status, body)
	}

	closeSession := func() error {
		endCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		status, body, err := gatewayRequest(endCtx, client, token, *base+"/api/sessions/"+*sessionID+"/end", "application/json", nil, 0)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("HTTP %d %s", status, body)
		}
		return nil
	}
	ended := false
	defer func() {
		if !ended {
			if err := closeSession(); err != nil {
				log.Printf("close session %s: %v", *sessionID, err)
			}
		}
	}()

	ffmpegArgs := []string{
		"-nostdin", "-hide_banner", "-loglevel", "error", "-i", *rtmpInput,
		"-map", "0:a:0", "-vn", "-ac", "1", "-ar", "16000",
		"-acodec", "pcm_s16le", "-f", "s16le", "pipe:1",
	}
	command := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	command.Stderr = os.Stderr
	pcm, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("start FFmpeg output: %w", err)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start FFmpeg: %w", err)
	}
	fmt.Printf("Session %s is receiving OBS audio. Overlay: %s/obs/%s\n", *sessionID, *base, *sessionID)
	fmt.Println("Stop OBS streaming or press Ctrl-C to close the session.")
	sequence, ingestErr := forwardPCM(pcm, func(sequence int64, chunk []byte) error {
		for attempt := 0; attempt < 30; attempt++ {
			status, body, err := gatewayRequest(ctx, client, token, *base+"/api/sessions/"+*sessionID+"/audio", "application/octet-stream", chunk, sequence)
			if err != nil {
				return fmt.Errorf("audio chunk %d delivery uncertain; inspect session before retry: %w", sequence, err)
			}
			if status == http.StatusAccepted {
				return nil
			}
			if status != http.StatusTooManyRequests {
				return fmt.Errorf("audio chunk %d: HTTP %d %s", sequence, status, body)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		}
		return errors.New("audio queue remained full for 6 seconds")
	})
	if ingestErr != nil {
		_ = command.Process.Kill()
	}
	waitErr := command.Wait()
	endErr := closeSession()
	ended = true
	if ingestErr != nil {
		return ingestErr
	}
	if waitErr != nil {
		return fmt.Errorf("FFmpeg stopped: %w", waitErr)
	}
	if sequence == 0 {
		return errors.New("OBS stream ended without audio; check that OBS sends its audio track")
	}
	if endErr != nil {
		return fmt.Errorf("close session: %w", endErr)
	}
	fmt.Printf("Session %s ended: %d audio chunks sent\n", *sessionID, sequence)
	return nil
}

func forwardPCM(source io.Reader, send func(int64, []byte) error) (int64, error) {
	buffer := make([]byte, audioChunkBytes)
	var sequence int64
	for {
		count, readErr := io.ReadFull(source, buffer)
		if count > 0 {
			if count%2 != 0 {
				return sequence, errors.New("FFmpeg returned an odd number of PCM bytes")
			}
			sequence++
			chunk := append([]byte(nil), buffer[:count]...)
			if err := send(sequence, chunk); err != nil {
				return sequence, err
			}
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			return sequence, nil
		}
		if readErr != nil {
			return sequence, readErr
		}
	}
}

func gatewayRequest(ctx context.Context, client *http.Client, token, target, contentType string, data []byte, sequence int64) (int, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(data))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", contentType)
	if sequence > 0 {
		request.Header.Set("X-Audio-Sequence", fmt.Sprint(sequence))
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8192))
	return response.StatusCode, body, err
}
