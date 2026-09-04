package model

import "benetnasch/app/domain/port"

type AgentMemoryAssertionDTO = port.AgentMemoryAssertionDTO
type AgentMemoryAssertionRevisionDTO = port.AgentMemoryAssertionRevisionDTO
type AgentMemoryConflictMemberDTO = port.AgentMemoryConflictMemberDTO
type AgentMemoryConflictDTO = port.AgentMemoryConflictDTO
type AgentMemoryAssertionPageDTO = port.AgentMemoryAssertionPageDTO
type AgentMemoryHistoryDTO = port.AgentMemoryHistoryDTO
type AgentMemoryConflictPageDTO = port.AgentMemoryConflictPageDTO

func NewAgentMemoryAssertionDTO(assertion port.AgentMemoryAssertion) AgentMemoryAssertionDTO {
	return port.NewAgentMemoryAssertionDTO(assertion)
}

func NewAgentMemoryAssertionRevisionDTO(revision port.AgentMemoryAssertionRevision) AgentMemoryAssertionRevisionDTO {
	return port.NewAgentMemoryAssertionRevisionDTO(revision)
}

func NewAgentMemoryConflictDTO(conflict port.AgentMemoryConflict) AgentMemoryConflictDTO {
	return port.NewAgentMemoryConflictDTO(conflict)
}
