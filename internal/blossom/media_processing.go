package blossom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/buckket/go-blurhash"
	"github.com/disintegration/imaging"
	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	storedb "github.com/gabrielmoura/nostr-relay-server/internal/db"
	jsonx "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/minio/sha256-simd"
	"github.com/tmthrgd/go-hex"
	"go.uber.org/zap"
)

type mediaOptimizationResult struct {
	width           *int32
	height          *int32
	durationMS      *int64
	bitrateKbps     *int32
	blurhash        string
	thumbnailHash   string
	optimizedHash   string
	hlsManifestHash string
	processingState string
	processingError string
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
	BitRate  string `json:"bit_rate"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	Width     int32  `json:"width"`
	Height    int32  `json:"height"`
	BitRate   string `json:"bit_rate"`
	Duration  string `json:"duration"`
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

func processMediaOptimization(ctx context.Context, object dbmodel.Object, filePath string) (mediaOptimizationResult, error) {
	processingCfg := config.Cfg.Store.MediaProcessing
	result := mediaOptimizationResult{processingState: "ready"}
	if !processingCfg.Enabled {
		return result, nil
	}

	switch {
	case strings.HasPrefix(object.MimeType, "image/"):
		return processImageOptimization(ctx, object, filePath)
	case strings.HasPrefix(object.MimeType, "video/"):
		return processAVOptimization(ctx, object, filePath, true)
	case strings.HasPrefix(object.MimeType, "audio/"):
		return processAVOptimization(ctx, object, filePath, false)
	default:
		return result, nil
	}
}

func processImageOptimization(ctx context.Context, object dbmodel.Object, filePath string) (mediaOptimizationResult, error) {
	processingCfg := config.Cfg.Store.MediaProcessing
	img, err := imaging.Open(filePath)
	if err != nil {
		return mediaOptimizationResult{}, fmt.Errorf("open image for optimization: %w", err)
	}
	result := mediaOptimizationResult{processingState: "ready"}
	bounds := img.Bounds()
	width := int32(bounds.Dx())
	height := int32(bounds.Dy())
	if processingCfg.ExtractMetadata {
		result.width = &width
		result.height = &height
	}

	optimizedImg := imaging.Fit(img, maxInt(processingCfg.Image.MaxWidth, 1600), maxInt(processingCfg.Image.MaxWidth, 1600), imaging.Lanczos)

	if processingCfg.Image.GenerateThumbnail {
		thumbnailPath, thumbErr := tempDerivativePath("thumbnail", ".webp")
		if thumbErr != nil {
			noteRecoverableMediaError(&result, thumbErr, "prepare image thumbnail path")
		} else {
			defer os.Remove(thumbnailPath)
			if err := runFFmpegImageThumbnailWebP(ctx, filePath, thumbnailPath, maxInt(processingCfg.Image.ThumbnailMaxWidth, 320)); err != nil {
				noteRecoverableMediaError(&result, err, "generate image thumbnail")
			} else if hash, err := persistDerivativeFromPath(ctx, object, thumbnailPath, "image/webp"); err != nil {
				noteRecoverableMediaError(&result, err, "persist image thumbnail")
			} else {
				result.thumbnailHash = hash
			}
		}
	}

	if processingCfg.Image.GenerateWebP {
		optimizedPath, optErr := tempDerivativePath("optimized", ".webp")
		if optErr != nil {
			noteRecoverableMediaError(&result, optErr, "prepare optimized image path")
		} else {
			defer os.Remove(optimizedPath)
			if err := runFFmpegImageWebP(ctx, filePath, optimizedPath, maxInt(processingCfg.Image.MaxWidth, 1600)); err != nil {
				noteRecoverableMediaError(&result, err, "generate optimized image webp")
			} else if hash, err := persistDerivativeFromPath(ctx, object, optimizedPath, "image/webp"); err != nil {
				noteRecoverableMediaError(&result, err, "persist optimized image webp")
			} else {
				result.optimizedHash = hash
			}
		}
	}

	if processingCfg.GenerateBlurhash {
		blurSource := optimizedImg
		if result.optimizedHash == "" {
			blurSource = imaging.Fit(img, 128, 128, imaging.Lanczos)
		}
		blurValue, err := blurhash.Encode(4, 3, imaging.Fit(blurSource, 128, 128, imaging.Lanczos))
		if err != nil {
			noteRecoverableMediaError(&result, err, "generate blurhash")
		} else {
			result.blurhash = blurValue
		}
	}

	return result, nil
}

func processAVOptimization(ctx context.Context, object dbmodel.Object, filePath string, withThumbnail bool) (mediaOptimizationResult, error) {
	processingCfg := config.Cfg.Store.MediaProcessing
	result := mediaOptimizationResult{processingState: "ready"}
	if processingCfg.ExtractMetadata {
		meta, err := probeMedia(ctx, filePath)
		if err != nil {
			noteRecoverableMediaError(&result, err, "extract media metadata")
		} else {
			result.width = meta.width
			result.height = meta.height
			result.durationMS = meta.durationMS
			result.bitrateKbps = meta.bitrateKbps
		}
	}

	if withThumbnail && processingCfg.Video.GenerateThumbnail {
		thumbnailPath, err := tempDerivativePath("thumb", ".webp")
		if err != nil {
			noteRecoverableMediaError(&result, err, "prepare video thumbnail path")
		} else {
			defer os.Remove(thumbnailPath)
			if err := runFFmpegThumbnailWebP(ctx, filePath, thumbnailPath, maxInt(processingCfg.Video.ThumbnailMaxWidth, 320)); err != nil {
				noteRecoverableMediaError(&result, err, "generate video thumbnail")
			} else if hash, err := persistDerivativeFromPath(ctx, object, thumbnailPath, "image/webp"); err != nil {
				noteRecoverableMediaError(&result, err, "persist video thumbnail")
			} else {
				result.thumbnailHash = hash
			}
		}
	}

	if withThumbnail && processingCfg.Video.GeneratePosterWebP {
		posterPath, err := tempDerivativePath("poster", ".webp")
		if err != nil {
			noteRecoverableMediaError(&result, err, "prepare video poster path")
		} else {
			defer os.Remove(posterPath)
			if err := runFFmpegPosterWebP(ctx, filePath, posterPath); err != nil {
				noteRecoverableMediaError(&result, err, "generate video poster")
			} else if hash, err := persistDerivativeFromPath(ctx, object, posterPath, "image/webp"); err != nil {
				noteRecoverableMediaError(&result, err, "persist video poster")
			} else {
				result.optimizedHash = hash
				if processingCfg.GenerateBlurhash {
					if blurValue, err := blurhashFromFile(posterPath); err != nil {
						noteRecoverableMediaError(&result, err, "generate video blurhash")
					} else {
						result.blurhash = blurValue
					}
				}
			}
		}
	}

	streamHash, err := processStreamingVariants(ctx, object, filePath)
	if err != nil {
		noteRecoverableMediaError(&result, err, "generate streaming manifest")
	} else if streamHash != "" {
		result.hlsManifestHash = streamHash
	}
	return result, nil
}

func processStreamingVariants(ctx context.Context, object dbmodel.Object, filePath string) (string, error) {
	processingCfg := config.Cfg.Store.MediaProcessing
	if !processingCfg.Streaming.EnableHLS && !processingCfg.Streaming.EnableDASH {
		return "", nil
	}
	primaryHash := ""
	if processingCfg.Streaming.EnableHLS {
		hash, err := generateHLSManifest(ctx, object, filePath)
		if err != nil {
			return "", err
		}
		if hash != "" {
			primaryHash = hash
		}
	}
	if processingCfg.Streaming.EnableDASH {
		hash, err := generateDASHManifest(ctx, object, filePath)
		if err != nil {
			if primaryHash != "" {
				log.Logger.Warn("secondary dash manifest generation failed", zap.Error(err))
				return primaryHash, nil
			}
			return "", err
		}
		if primaryHash == "" {
			primaryHash = hash
		}
	}
	return primaryHash, nil
}

type mediaProbeResult struct {
	width       *int32
	height      *int32
	durationMS  *int64
	bitrateKbps *int32
}

func probeMedia(ctx context.Context, filePath string) (mediaProbeResult, error) {
	cmd := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration,bit_rate:stream=codec_type,width,height,bit_rate,duration",
		"-of", "json",
		filePath,
	)
	output, err := cmd.Output()
	if err != nil {
		return mediaProbeResult{}, fmt.Errorf("run ffprobe: %w", err)
	}
	var probed ffprobeOutput
	if err := json.Unmarshal(output, &probed); err != nil {
		return mediaProbeResult{}, fmt.Errorf("decode ffprobe output: %w", err)
	}
	result := mediaProbeResult{}
	for _, stream := range probed.Streams {
		if stream.Width > 0 && stream.Height > 0 && result.width == nil && result.height == nil {
			w := stream.Width
			h := stream.Height
			result.width = &w
			result.height = &h
		}
		if result.bitrateKbps == nil {
			if bitrate := parseBitrateKbps(stream.BitRate); bitrate != nil {
				result.bitrateKbps = bitrate
			}
		}
		if result.durationMS == nil {
			if duration := parseDurationMS(stream.Duration); duration != nil {
				result.durationMS = duration
			}
		}
	}
	if result.bitrateKbps == nil {
		result.bitrateKbps = parseBitrateKbps(probed.Format.BitRate)
	}
	if result.durationMS == nil {
		result.durationMS = parseDurationMS(probed.Format.Duration)
	}
	return result, nil
}

func runFFmpegThumbnail(ctx context.Context, inputPath string, outputPath string) error {
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", inputPath,
		"-vf", "thumbnail,scale='min(320,iw)':-2",
		"-frames:v", "1",
		outputPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("run ffmpeg thumbnail: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runFFmpegThumbnailWebP(ctx context.Context, inputPath string, outputPath string, maxWidth int) error {
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("thumbnail,scale='min(%d,iw)':-2", maxWidth),
		"-frames:v", "1",
		"-q:v", "75",
		outputPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("run ffmpeg thumbnail webp: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runFFmpegImageWebP(ctx context.Context, inputPath string, outputPath string, maxWidth int) error {
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale='min(%d,iw)':-2", maxWidth),
		"-q:v", "75",
		outputPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("run ffmpeg webp conversion: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runFFmpegImageThumbnailWebP(ctx context.Context, inputPath string, outputPath string, maxWidth int) error {
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale='min(%d,iw)':-2", maxWidth),
		"-q:v", "75",
		outputPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("run ffmpeg image thumbnail webp: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runFFmpegPosterWebP(ctx context.Context, inputPath string, outputPath string) error {
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", inputPath,
		"-vf", "thumbnail,scale='min(1280,iw)':-2",
		"-frames:v", "1",
		"-q:v", "75",
		outputPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("run ffmpeg poster webp: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func persistImageDerivative(ctx context.Context, object dbmodel.Object, img image.Image, name string, quality int) (string, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return "", fmt.Errorf("encode %s derivative: %w", name, err)
	}
	return persistDerivativeBytes(ctx, object, buf.Bytes(), "image/jpeg")
}

func persistDerivativeFromPath(ctx context.Context, object dbmodel.Object, path string, mimeType string) (string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read derivative file: %w", err)
	}
	return persistDerivativeBytes(ctx, object, payload, mimeType)
}

func persistDerivativeBytes(ctx context.Context, object dbmodel.Object, payload []byte, mimeType string) (string, error) {
	hasher := sha256.New()
	_, _ = hasher.Write(payload)
	hash := hex.EncodeToString(hasher.Sum(nil))
	path := filepath.Join(blobPath, hash)
	if err := os.WriteFile(path, payload, 0o644); err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("persist derivative payload: %w", err)
	}
	derivedObject := &dbmodel.Object{
		Hash:      hash,
		CreatedAt: object.CreatedAt,
		MimeType:  mimeType,
		Size:      int64(len(payload)),
		PublicKey: object.PublicKey,
	}
	if err := storedb.DbQueries.InsertObject(ctx, derivedObject); err != nil {
		return "", fmt.Errorf("insert derivative object: %w", err)
	}
	return hash, nil
}

func tempDerivativePath(prefix string, ext string) (string, error) {
	file, err := os.CreateTemp(blobPath, prefix+"-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp derivative: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close temp derivative: %w", err)
	}
	return path, nil
}

func parseDurationMS(value string) *int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	seconds, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil
	}
	ms := int64(seconds * 1000)
	return &ms
}

func parseBitrateKbps(value string) *int32 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	bits, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return nil
	}
	kbps := int32(bits / 1000)
	return &kbps
}

func marshalNIP94Tags(tags [][]string) []byte {
	payload, err := jsonx.Marshal(tags)
	if err != nil {
		return []byte("[]")
	}
	return payload
}

func decodeMirrorURLsFromPayload(payload []byte) []string {
	var mirrors []string
	if len(payload) == 0 {
		return []string{}
	}
	if err := json.Unmarshal(payload, &mirrors); err != nil {
		return []string{}
	}
	return uniqueStrings(mirrors)
}

func blurhashFromFile(path string) (string, error) {
	img, err := imaging.Open(path)
	if err != nil {
		return "", fmt.Errorf("open image for blurhash: %w", err)
	}
	return blurhash.Encode(4, 3, imaging.Fit(img, 128, 128, imaging.Lanczos))
}

func noteRecoverableMediaError(result *mediaOptimizationResult, err error, operation string) {
	if err == nil {
		return
	}
	message := fmt.Sprintf("%s: %v", operation, err)
	log.Logger.Warn("blossom media processing step failed", zap.String("operation", operation), zap.Error(err))
	if strings.TrimSpace(result.processingError) == "" {
		result.processingError = message
	} else {
		result.processingError = result.processingError + "; " + message
	}
}

func maxInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func generateHLSManifest(ctx context.Context, object dbmodel.Object, filePath string) (string, error) {
	dir, err := os.MkdirTemp(blobPath, "hls-*")
	if err != nil {
		return "", fmt.Errorf("create hls temp dir: %w", err)
	}
	defer os.RemoveAll(dir)
	manifestPath := filepath.Join(dir, "stream.m3u8")
	segmentPath := filepath.Join(dir, "segment.ts")
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", filePath,
		"-c", "copy",
		"-f", "hls",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_flags", "single_file",
		"-hls_segment_filename", segmentPath,
		manifestPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("run ffmpeg hls: %w: %s", err, strings.TrimSpace(string(output)))
	}
	replacements := map[string]string{}
	segmentHash, err := persistGeneratedArtifact(ctx, object, segmentPath)
	if err != nil {
		return "", err
	}
	replacements[filepath.Base(segmentPath)] = directURL(segmentHash)
	return persistRewrittenManifest(ctx, object, manifestPath, replacements, "application/vnd.apple.mpegurl")
}

func generateDASHManifest(ctx context.Context, object dbmodel.Object, filePath string) (string, error) {
	dir, err := os.MkdirTemp(blobPath, "dash-*")
	if err != nil {
		return "", fmt.Errorf("create dash temp dir: %w", err)
	}
	defer os.RemoveAll(dir)
	manifestPath := filepath.Join(dir, "stream.mpd")
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", filePath,
		"-map", "0",
		"-c", "copy",
		"-f", "dash",
		"-seg_duration", "6",
		"-single_file", "1",
		manifestPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("run ffmpeg dash: %w: %s", err, strings.TrimSpace(string(output)))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read dash temp dir: %w", err)
	}
	replacements := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == filepath.Base(manifestPath) {
			continue
		}
		artifactPath := filepath.Join(dir, entry.Name())
		hash, err := persistGeneratedArtifact(ctx, object, artifactPath)
		if err != nil {
			return "", err
		}
		replacements[entry.Name()] = directURL(hash)
	}
	return persistRewrittenManifest(ctx, object, manifestPath, replacements, "application/dash+xml")
}

func persistGeneratedArtifact(ctx context.Context, object dbmodel.Object, path string) (string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read generated artifact: %w", err)
	}
	return persistDerivativeBytes(ctx, object, payload, artifactMIMEType(path))
}

func persistRewrittenManifest(ctx context.Context, object dbmodel.Object, manifestPath string, replacements map[string]string, mimeType string) (string, error) {
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("read manifest: %w", err)
	}
	content := string(payload)
	for name, replacement := range replacements {
		content = strings.ReplaceAll(content, name, replacement)
	}
	return persistDerivativeBytes(ctx, object, []byte(content), mimeType)
}

func artifactMIMEType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".m3u8":
		return "application/vnd.apple.mpegurl"
	case ".mpd":
		return "application/dash+xml"
	case ".ts":
		return "video/mp2t"
	case ".m4s", ".mp4":
		return "video/mp4"
	case ".webp":
		return "image/webp"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}
