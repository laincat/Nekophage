package core

import (
	"github.com/laincat/veneracat-core/aiapi"
	"github.com/laincat/veneracat-core/gohost"
	"github.com/laincat/veneracat-core/novel/adapt"
)

// Wave 4B static comic adaptation and rendering types exposed from VeneraCat Core.
// (*Core).AdaptChapter, (*Core).RenderPanel and (*Core).RenderStoryboard are
// inherited directly from gohost.Core.
type (
	AIProtocol = aiapi.Protocol

	AdaptTextConfig         = gohost.AdaptTextConfig
	AdaptImageProvider      = gohost.AdaptImageProvider
	AdaptImageConfig        = gohost.AdaptImageConfig
	AdaptChapterRequest     = gohost.AdaptChapterRequest
	RenderPanelRequest      = gohost.RenderPanelRequest
	RenderStoryboardRequest = gohost.RenderStoryboardRequest

	AdaptProject           = adapt.Project
	AdaptStyleRef          = adapt.StyleRef
	AdaptCharacter         = adapt.Character
	AdaptCharacterSheet    = adapt.CharacterSheet
	AdaptDialogue          = adapt.Dialogue
	AdaptPanel             = adapt.Panel
	AdaptScene             = adapt.Scene
	AdaptStoryboard        = adapt.Storyboard
	AdaptBudget            = adapt.Budget
	AdaptUsage             = adapt.Usage
	AdaptOptions           = adapt.AdaptOptions
	AdaptCheckpoint        = adapt.AdaptCheckpoint
	AdaptResult            = adapt.AdaptResult
	AdaptImageStatus       = adapt.ImageStatus
	AdaptImageRefKind      = adapt.ImageRefKind
	AdaptImageRef          = adapt.ImageRef
	AdaptRenderOptions     = adapt.RenderOptions
	AdaptPanelFailure      = adapt.PanelFailure
	AdaptPanelRenderResult = adapt.PanelRenderResult
	AdaptRenderResult      = adapt.RenderResult
	AdaptPanelImageMeta    = adapt.PanelImageMeta
	AdaptPanelImageRequest = adapt.PanelImageRequest
	AdaptPanelImageResult  = adapt.PanelImageResult
	AdaptTextGenerator     = adapt.TextGenerator
	AdaptImageGenerator    = adapt.ImageGenerator
	AdaptImageSink         = adapt.ImageSink
	AdaptImageSinkFunc     = adapt.ImageSinkFunc
	AdaptGenerateRequest   = adapt.GenerateRequest
	AdaptGenerateResponse  = adapt.GenerateResponse
)

const (
	AIProtocolChat      = aiapi.Chat
	AIProtocolResponses = aiapi.Responses
	AIProtocolMessages  = aiapi.Messages

	AdaptImageGemini = gohost.AdaptImageGemini
	AdaptImageOpenAI = gohost.AdaptImageOpenAI

	AdaptImagePending    = adapt.ImagePending
	AdaptImageGenerating = adapt.ImageGenerating
	AdaptImageDone       = adapt.ImageDone
	AdaptImageFailed     = adapt.ImageFailed

	AdaptImageRefStyle     = adapt.ImageRefStyle
	AdaptImageRefCharacter = adapt.ImageRefCharacter
)

var (
	ErrAdaptBudgetExceeded    = adapt.ErrBudgetExceeded
	ErrAdaptInvalidStoryboard = adapt.ErrInvalidStoryboard
	ErrAdaptEmptyChapter      = adapt.ErrEmptyChapter
	ErrAdaptContentBlocked    = adapt.ErrContentBlocked
)
