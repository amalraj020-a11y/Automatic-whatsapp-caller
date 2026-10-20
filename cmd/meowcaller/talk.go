package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	meowcaller "github.com/purpshell/meowcaller"
	"github.com/rs/zerolog"
)

type talkConfig struct {
	target, storePath, transcriber, tempDir, ollamaURL, model string
	answerTimeout                                             time.Duration
}

func parseTalkArgs(args []string, stderr io.Writer) (talkConfig, error) {
	var cfg talkConfig
	store, err := defaultStorePath()
	if err != nil {
		return cfg, err
	}
	f := flag.NewFlagSet("talk", flag.ContinueOnError)
	f.SetOutput(stderr)
	f.StringVar(&cfg.storePath, "store", store, "linked-device SQLite store")
	f.StringVar(&cfg.transcriber, "transcriber", os.Getenv("AMALA_TRANSCRIBER"), "PowerShell local Whisper wrapper")
	f.StringVar(&cfg.tempDir, "temp-dir", os.TempDir(), "private temporary audio directory")
	f.StringVar(&cfg.ollamaURL, "ollama-url", "http://127.0.0.1:11434/api/generate", "local Ollama generate endpoint")
	f.StringVar(&cfg.model, "model", "qwen3:4b", "local Ollama model")
	f.DurationVar(&cfg.answerTimeout, "answer-timeout", 45*time.Second, "maximum wait for answer")
	if err := f.Parse(args); err != nil {
		return cfg, errors.Join(errUsage, err)
	}
	if f.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: meowcaller talk [flags] <target>")
		return cfg, errUsage
	}
	cfg.target = f.Arg(0)
	if cfg.transcriber == "" {
		return cfg, errors.New("--transcriber or AMALA_TRANSCRIBER is required")
	}
	return cfg, nil
}

type vadSink struct {
	mu           sync.Mutex
	speech       bool
	reported     bool
	silent       int
	samples      []float32
	out          chan<- []float32
	log          io.Writer
	frames       uint64
	speechStarts uint64
	clips        uint64
	peakRMS      float64
}

const vadThreshold = 0.008
const vadSilenceFrames = 25 // 500 ms at 20 ms per decoded audio frame.

func (v *vadSink) Close() error { return nil }
func (v *vadSink) WriteFrame(frame []float32) error {
	if len(frame) == 0 {
		return nil
	}
	var sum float64
	for _, x := range frame {
		sum += float64(x * x)
	}
	rms := math.Sqrt(sum / float64(len(frame)))
	v.mu.Lock()
	defer v.mu.Unlock()
	v.frames++
	if rms > v.peakRMS {
		v.peakRMS = rms
	}
	if rms > vadThreshold {
		if !v.speech && !v.reported && v.log != nil {
			fmt.Fprintf(v.log, "Inbound speech-level frame detected (RMS %.4f)\n", rms)
			v.reported = true
		}
		if !v.speech {
			v.speechStarts++
		}
		v.speech = true
		v.silent = 0
	}
	if v.speech {
		v.samples = append(v.samples, frame...)
		if rms <= vadThreshold {
			v.silent++
		}
	}
	if v.speech && v.silent >= vadSilenceFrames {
		clip := append([]float32(nil), v.samples...)
		v.samples = nil
		v.speech = false
		v.reported = false
		v.silent = 0
		if len(clip) >= meowcaller.SampleRate/2 {
			v.clips++
			if v.log != nil {
				fmt.Fprintf(v.log, "Inbound speech clip captured: %.1f seconds\n", float64(len(clip))/float64(meowcaller.SampleRate))
			}
			select {
			case v.out <- clip:
			default:
				if v.log != nil {
					fmt.Fprintln(v.log, "Inbound speech clip dropped: processing queue is full")
				}
			}
		}
	}
	return nil
}

func (v *vadSink) summary() (frames, starts, clips uint64, peak float64, pendingSamples int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.frames, v.speechStarts, v.clips, v.peakRMS, len(v.samples)
}

func runTalk(ctx context.Context, cfg talkConfig, stderr io.Writer) error {
	log := zerolog.Ctx(ctx)
	session, err := openSession(ctx, cfg.storePath, *log)
	if err != nil {
		return err
	}
	defer session.Close()
	client := meowcaller.NewClient(session.client, meowcaller.WithLogger(*log))
	if err = session.ConnectPaired(ctx); err != nil {
		return err
	}
	call, err := client.Call(ctx, cfg.target)
	if err != nil {
		return fmt.Errorf("place call: %w", err)
	}
	ready := make(chan struct{}, 1)
	answered := make(chan struct{}, 1)
	ended := make(chan string, 1)
	// Local Whisper + Ollama can take longer than the user's pause between phrases.
	// Buffer several completed phrases so active decoding does not immediately drop them.
	utterances := make(chan []float32, 8)
	voice := &vadSink{out: utterances, log: stderr}
	call.Receive(voice)
	player := meowcaller.NewPlayer()
	call.Subscribe(player)
	call.OnAnswered(func() {
		select {
		case answered <- struct{}{}:
		default:
		}
	})
	registerNotificationCallbacks(call,
		func() {
			select {
			case ready <- struct{}{}:
			default:
			}
		},
		func(reason string) {
			select {
			case ended <- reason:
			default:
			}
		},
	)
	answerTimer := time.NewTimer(cfg.answerTimeout)
	defer answerTimer.Stop()
	// Wait for WhatsApp's explicit CallAccept event, rather than guessing that
	// an elapsed ring delay means the phone was answered. The registered callback
	// replays the event immediately when the accept beat us to registration.
	select {
	case <-answered:
		fmt.Fprintln(stderr, "WhatsApp accepted the call; starting Amala's greeting")
	case r := <-ended:
		return fmt.Errorf("call ended before greeting: %s", r)
	case <-answerTimer.C:
		_ = call.Hangup()
		return fmt.Errorf("no WhatsApp answer event within %s", cfg.answerTimeout)
	case <-ctx.Done():
		return ctx.Err()
	}
	fmt.Fprintln(stderr, "Amala outbound audio started; waiting for speech")
	if src, e := synthesize(ctx, cfg.tempDir, "Hello Raj. This is Amala. I can hear you. Please speak."); e == nil {
		player.OnFinish(func() {
			fmt.Fprintln(stderr, "Greeting audio source finished locally; remote audibility is not confirmed")
		})
		player.Play(src)
		fmt.Fprintln(stderr, "Greeting audio queued to the call media sender")
	} else {
		fmt.Fprintln(stderr, "greeting tts failed:", e)
	}
	for {
		select {
		case <-ready:
			fmt.Fprintln(stderr, "Inbound WhatsApp audio is flowing")
		case clip := <-utterances:
			// Give a short local spoken acknowledgement before CPU Whisper starts;
			// otherwise several seconds of silence can sound like a dropped call.
			ack := "I heard you. Give me a moment to think."
			if src, err := synthesize(ctx, cfg.tempDir, ack); err == nil {
				player.Play(src)
				fmt.Fprintln(stderr, "Amala acknowledgement queued; preparing transcription")
			} else {
				fmt.Fprintln(stderr, "acknowledgement tts failed:", err)
			}
			wav := filepath.Join(cfg.tempDir, fmt.Sprintf("amala-in-%d.wav", time.Now().UnixNano()))
			if err := writeWAV(wav, clip); err != nil {
				fmt.Fprintln(stderr, err)
				continue
			}
			text, err := transcribe(ctx, cfg.transcriber, wav)
			_ = os.Remove(wav)
			if err != nil || strings.TrimSpace(text) == "" {
				fmt.Fprintln(stderr, "transcription failed:", err)
				continue
			}
			fmt.Fprintln(stderr, "Raj:", text)
			reply, err := reason(ctx, cfg, text)
			if err != nil {
				reply = "I heard you, but my local reasoning service is unavailable."
				fmt.Fprintln(stderr, err)
			}
			fmt.Fprintln(stderr, "Amala:", reply)
			src, err := synthesize(ctx, cfg.tempDir, reply)
			if err == nil {
				player.Play(src)
			} else {
				fmt.Fprintln(stderr, err)
			}
		case r := <-ended:
			frames, starts, clips, peak, pending := voice.summary()
			fmt.Fprintf(stderr, "Call audio summary: decoded_frames=%d speech_onsets=%d completed_clips=%d peak_rms=%.4f pending_samples=%d\n", frames, starts, clips, peak, pending)
			fmt.Fprintln(stderr, "caller ended:", r)
			return nil
		case <-ctx.Done():
			_ = call.Hangup()
			return ctx.Err()
		}
	}
}

func writeWAV(path string, s []float32) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	n := uint32(len(s) * 2)
	h := bytes.NewBuffer(nil)
	h.WriteString("RIFF")
	binary.Write(h, binary.LittleEndian, uint32(36)+n)
	h.WriteString("WAVEfmt ")
	binary.Write(h, binary.LittleEndian, uint32(16))
	binary.Write(h, binary.LittleEndian, uint16(1))
	binary.Write(h, binary.LittleEndian, uint16(1))
	binary.Write(h, binary.LittleEndian, uint32(16000))
	binary.Write(h, binary.LittleEndian, uint32(32000))
	binary.Write(h, binary.LittleEndian, uint16(2))
	binary.Write(h, binary.LittleEndian, uint16(16))
	h.WriteString("data")
	binary.Write(h, binary.LittleEndian, n)
	if _, e = f.Write(h.Bytes()); e != nil {
		return e
	}
	for _, x := range s {
		if x > 1 {
			x = 1
		}
		if x < -1 {
			x = -1
		}
		binary.Write(f, binary.LittleEndian, int16(x*32767))
	}
	return nil
}

func transcribe(ctx context.Context, script, wav string) (string, error) {
	c := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script, wav)
	b, e := c.CombinedOutput()
	return strings.TrimSpace(string(b)), e
}

func reason(ctx context.Context, cfg talkConfig, text string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": cfg.model, "stream": false, "think": false,
		"prompt":  "You are Amala speaking to Raj on a phone call. Reply naturally in the same Tamil, English, or Tanglish style in one concise sentence (under 16 words). Never claim an action was completed unless verified. Raj said: " + text,
		"options": map[string]any{"num_predict": 48},
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", cfg.ollamaURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("ollama status %s", resp.Status)
	}
	var out struct {
		Response string `json:"response"`
	}
	if e = json.NewDecoder(resp.Body).Decode(&out); e != nil {
		return "", e
	}
	if strings.TrimSpace(out.Response) == "" {
		return "", errors.New("local reasoning returned no spoken response")
	}
	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	return strings.TrimSpace(re.ReplaceAllString(out.Response, "")), nil
}

func synthesize(ctx context.Context, dir, text string) (meowcaller.AudioSource, error) {
	path := filepath.Join(dir, fmt.Sprintf("amala-out-%d.wav", time.Now().UnixNano()))
	// Use -File with named parameters. Passing free-form speech text after
	// PowerShell -Command makes PowerShell parse the words as source code.
	psPath := filepath.Join(dir, "amala-sapi-tts.ps1")
	ps := `param([Parameter(Mandatory=$true)][string]$Text,[Parameter(Mandatory=$true)][string]$Path,[Parameter(Mandatory=$true)][string]$VoiceToken)
$ErrorActionPreference = 'Stop'
$voice = New-Object -ComObject SAPI.SpVoice
$token = New-Object -ComObject SAPI.SpObjectToken
$token.SetId($VoiceToken)
$voice.Voice = $token
$stream = New-Object -ComObject SAPI.SpFileStream
try {
    $stream.Open($Path, 3, $false)
    $voice.AudioOutputStream = $stream
    [void]$voice.Speak($Text)
} finally {
    $stream.Close()
}`
	if err := os.WriteFile(psPath, []byte(ps), 0600); err != nil {
		return nil, fmt.Errorf("write tts script: %w", err)
	}
	voiceToken := `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Speech\Voices\Tokens\TTS_MS_EN-US_ZIRA_11.0`
	c := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", psPath, "-Text", text, "-Path", path, "-VoiceToken", voiceToken)
	if b, e := c.CombinedOutput(); e != nil {
		return nil, fmt.Errorf("tts: %w: %s", e, b)
	}
	return meowcaller.WAVFile(path)
}
