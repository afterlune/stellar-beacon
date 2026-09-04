package model

import "benetnasch/app/domain/port"

type AIWritingPreviewDTO = port.AIWritingPreviewDTO
type AIVisionPreviewDTO = port.AIVisionPreviewDTO
type AIReviewDTO = port.AIReviewDTO
type AgentProfileDTO = port.AgentProfileDTO
type AgentProfileUpdateVO = port.AgentProfileUpdateVO
type AgentReviewPolicyDTO = port.AgentReviewPolicyDTO
type AgentReviewPolicyUpdateVO = port.AgentReviewPolicyUpdateVO
type ContentGalaxyPointDTO = port.ContentGalaxyPointDTO
type ContentGalaxyPageDTO = port.ContentGalaxyPageDTO
type DreamDTO = port.DreamDTO
type DreamPageDTO = port.DreamPageDTO
type TimeCapsuleDTO = port.TimeCapsuleDTO
type RadioEpisodeDTO = port.RadioEpisodeDTO
type RadioPageDTO = port.RadioPageDTO
type VideoDTO = port.VideoDTO
type VideoPageDTO = port.VideoPageDTO

func NewAIReviewDTO(review port.AIReview) AIReviewDTO {
	return port.NewAIReviewDTO(review)
}
