package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"unihub-workshop/internal/circuitbreaker"
	"unihub-workshop/internal/repository"

	"github.com/google/generative-ai-go/genai"
	"github.com/ledongthuc/pdf"
	"google.golang.org/api/option"
)

// AISummaryService implements Pipe-and-Filter architecture for PDF summarization
type AISummaryService struct {
	workshopRepo *repository.WorkshopRepo
	breaker      *circuitbreaker.CircuitBreaker
	apiKey       string
	modelName    string
	temperature  float64
	maxTokens    int
}

// NewAISummaryService creates a new instance of AISummaryService
func NewAISummaryService(workshopRepo *repository.WorkshopRepo, apiKey, modelName string, temperature float64, maxTokens int) *AISummaryService {
	if modelName == "" {
		modelName = "gemini-2.0-flash"
	}
	return &AISummaryService{
		workshopRepo: workshopRepo,
		breaker:      circuitbreaker.NewCircuitBreaker("ai-service", 0.5, 30*time.Second, 60*time.Second),
		apiKey:       apiKey,
		modelName:    modelName,
		temperature:  temperature,
		maxTokens:    maxTokens,
	}
}

// ProcessPDF implements the full pipeline including persistence (for backward compatibility)
func (s *AISummaryService) ProcessPDF(ctx context.Context, workshopID string, pdfData []byte) (string, error) {
	log.Printf("[AI_SUMMARY] Starting full pipeline for workshop %s", workshopID)
	
	summary, err := s.Summarize(ctx, pdfData)
	if err != nil {
		return "", err
	}

	// Sink: Persist to database
	if s.workshopRepo != nil {
		if err := s.workshopRepo.UpdateSummary(ctx, workshopID, summary); err != nil {
			return "", fmt.Errorf("failed to persist summary: %w", err)
		}
	}

	log.Printf("[AI_SUMMARY] Full pipeline completed for workshop %s", workshopID)
	return summary, nil
}

// Summarize runs the pipeline and returns the summary without saving to DB
func (s *AISummaryService) Summarize(ctx context.Context, pdfData []byte) (string, error) {
	// Filter 1: Extract text from PDF
	rawText, err := s.extractText(pdfData)
	if err != nil {
		return "", fmt.Errorf("extraction failed: %w", err)
	}

	// Filter 2: Clean and normalize text
	cleanedText := s.cleanText(rawText)
	if cleanedText == "" {
		return "", fmt.Errorf("no readable text found in PDF after cleaning")
	}

	// Filter 3: Call AI model via Circuit Breaker
	var summary string
	err = s.breaker.Execute(func() error {
		var aiErr error
		summary, aiErr = s.callGemini(ctx, cleanedText)
		return aiErr
	})

	if err != nil {
		if err == circuitbreaker.ErrCircuitOpen {
			return "", fmt.Errorf("dịch vụ AI đang tạm thời gián đoạn (Circuit Breaker)")
		}
		return "", fmt.Errorf("AI summary failed: %w", err)
	}

	return summary, nil
}

// Filter 1: Extract text using ledongthuc/pdf
func (s *AISummaryService) extractText(pdfData []byte) (string, error) {
	reader := bytes.NewReader(pdfData)
	f, err := pdf.NewReader(reader, int64(len(pdfData)))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	b, err := f.GetPlainText()
	if err != nil {
		return "", err
	}
	
	_, err = io.Copy(&buf, b)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// cleanMarkdown strips common markdown symbols to keep the text clean for plain-text display
func (s *AISummaryService) cleanMarkdown(text string) string {
	// Remove bold/italic markers and headers
	replacer := strings.NewReplacer(
		"**", "",
		"__", "",
		"*", "",
		"###", "",
		"##", "",
		"#", "",
	)
	text = replacer.Replace(text)
	return strings.TrimSpace(text)
}

// Filter 2: Clean and normalize text
func (s *AISummaryService) cleanText(text string) string {
	// Remove extra whitespace and special characters
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")
	
	// Remove non-printable characters except common punctuation
	text = strings.Map(func(r rune) rune {
		if r >= 32 && r < 127 || r == '\n' || r == '\t' {
			return r
		}
		// Basic Vietnamese character support (simple range, not exhaustive)
		if r >= 0x00C0 && r <= 0x1EF9 {
			return r
		}
		return -1
	}, text)

	text = strings.TrimSpace(text)

	// Limit to ~15000 characters for Gemini Lite/Flash context efficiency
	if len(text) > 15000 {
		text = text[:15000]
	}

	return text
}

// Filter 3: Call Google Gemini API
func (s *AISummaryService) callGemini(ctx context.Context, text string) (string, error) {
	if s.apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY is not configured")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(s.apiKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(s.modelName)
	
	// Configure generation from service fields
	temp := float32(s.temperature)
	model.Temperature = &temp
	topP := float32(0.8)
	model.TopP = &topP
	topK := int32(40)
	model.TopK = &topK
	maxTokens := int32(s.maxTokens)
	model.MaxOutputTokens = &maxTokens

	// System instruction + Prompt
	prompt := []genai.Part{
		genai.Text(`Bạn là chuyên gia truyền thông và học thuật tại UniHub. Nhiệm vụ của bạn là đọc đề án Workshop từ file PDF và viết một bản tóm tắt chuyên nghiệp, lôi cuốn sinh viên.

Yêu cầu bản tóm tắt:
1. Không sử dụng bất kỳ định dạng Markdown nào (KHÔNG dùng **, *, #, __).
2. Tiêu đề: Viết hoa toàn bộ chủ đề cốt lõi.
3. Mục tiêu: Nêu rõ 3-4 giá trị thực chiến, sử dụng gạch đầu dòng "-" đơn giản.
4. Nội dung chính: Trình bày lộ trình dưới dạng danh sách gạch đầu dòng "-".
5. Đối tượng & Chuẩn bị: Ghi rõ ai nên tham gia và cần mang theo gì.
6. Ngôn ngữ: Tiếng Việt, văn phong hiện đại, sạch sẽ, chỉ dùng văn bản thuần túy (Plain Text).

Nội dung trích xuất từ PDF:
`),
		genai.Text(text),
	}

	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("AI không trả về nội dung phù hợp")
	}

	var summaryParts []string
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			summaryParts = append(summaryParts, string(textPart))
		}
	}

	return s.cleanMarkdown(strings.Join(summaryParts, "")), nil
}
